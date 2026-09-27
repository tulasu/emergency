package main

import "strings"

#Config: {
	root: *"/" | string

	postgres: {
		user:     string & =~"^.+$"
		password: string & =~"^.+$"
		database: string & =~"^.+$"
	}

	ticketgen: {
		llm: {
			backends: [...{
				name:     string
				url:      string
				model:    *"" | string
				priority: *0 | int
				timeout:  *60 | number // seconds
			}]
			max_retries: *3 | int
		}
		poll_seconds:  *2 | number
		lease_seconds: *120 | int
	}

	// single entrypoint: shared secrets/wiring for compose (dev.cue holds dev values)
	service_token: string & =~"^.+$"
	ari: {
		user:     string & =~"^.+$"
		password: string & =~"^.+$"
	}
	admin: {
		login:    string & =~"^.+$"
		password: string & =~"^.+$"
	}
	dialog: {
		device: *"cpu" | string
	}
	audio: {
		postgres: {
			user:     string & =~"^.+$"
			password: string & =~"^.+$"
			database: string & =~"^.+$"
		}
		s3: {
			access_key: string & =~"^.+$"
			secret_key: string & =~"^.+$"
		}
		voice: *"kseniya" | string
		rate:  *8000 | int
		synth_device: *"cuda" | string
		sweep_minutes: *10 | int
		traineebox_audio_url: *"http://audio:8002" | string
		dialog_audio_url: *"http://127.0.0.1:8002" | string
		traineebox_url: *"http://traineebox:8080" | string
	}
}

config: #Config

_caddyfile:         "\(strings.TrimSuffix(config.root, "/"))/etc/caddy/Caddyfile"
_catalog:           "\(strings.TrimSuffix(config.root, "/"))/etc/traineebox/catalog"
_audioDatabaseURL: "postgres://\(config.audio.postgres.user):\(config.audio.postgres.password)@audio-postgres:5432/\(config.audio.postgres.database)?sslmode=disable"

_llmBackendsEnv: "[" + strings.Join([ for b in config.ticketgen.llm.backends {
	#"{"name":"\#(b.name)","url":"\#(b.url)","model":"\#(b.model)","priority":\#(b.priority),"timeout":\#(b.timeout)}"#
}], ",") + "]"

compose: {
	services: {
		postgres: {
			image: "postgres:16"
			environment: {
				POSTGRES_USER:     config.postgres.user
				POSTGRES_PASSWORD: config.postgres.password
				POSTGRES_DB:       config.postgres.database
			}
			ports: ["5432:5432"]
			volumes: [
				"postgres_data:/var/lib/postgresql/data",
			]
		healthcheck: {
			test: ["CMD-SHELL", "pg_isready -U \(config.postgres.user)"]
			interval: "5s"
			timeout:  "3s"
			retries:   5
		}
			restart: "unless-stopped"
		}
		caddy: {
			image: "caddy:2"
			ports: ["80:80", "443:443"]
			volumes: [
				"\(_caddyfile):/etc/caddy/Caddyfile:ro",
				"caddy_data:/data",
				"caddy_config:/config",
			]
			restart: "unless-stopped"
		}
		llama: {
			image: "ghcr.io/ggml-org/llama.cpp:server-cuda"
			gpus:  "all"
			ports: ["8081:8081"]
			volumes: [
				"./artifacts/models/llama:/models:ro",
			]
			command: [
				"-m", "/models/Qwen3-1.7B-Q8_0.gguf",
				"-ngl", "99",
				"-c", "4096",
				"-np", "2",
				"--host", "0.0.0.0",
				"--port", "8081",
				"--no-webui",
			]
			// образ чекает :8080 (веб-UI по умолчанию), а мы подняли :8081
			healthcheck: {
				test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
				interval: "30s"
				timeout:  "3s"
				retries:   5
			}
			restart: "unless-stopped"
		}
		ticketgen: {
			build: {
				context:    "."
				dockerfile: "ticketgen/Dockerfile"
			}
			depends_on: ["postgres"]
			// host.docker.internal -> host-gateway: на Linux ticketgen тянет LLM
			// с http://host.docker.internal:8081 (см. config/dev.cue backends).
			extra_hosts: ["host.docker.internal:host-gateway"]
			environment: {
				TICKETGEN_POSTGRES_DSN:  "postgres://\(config.postgres.user):\(config.postgres.password)@postgres:5432/\(config.postgres.database)?sslmode=disable"
				TICKETGEN_CATALOG_PATH:  "/catalog"
				DIALOG_URL:              "http://dialog:8000"
				TICKETGEN_LLM_BACKENDS:  _llmBackendsEnv
				TICKETGEN_POLL_SECONDS:  "\(config.ticketgen.poll_seconds)"
				TICKETGEN_LEASE_SECONDS: "\(config.ticketgen.lease_seconds)"
			}
			volumes: [
				"\(_catalog):/catalog:ro",
			]
			restart: "unless-stopped"
		}
		traineebox: {
			build: {
				context:    "."
				dockerfile: "traineebox/Dockerfile"
			}
			depends_on: {
				postgres: {
					condition: "service_healthy"
				}
			}
			ports: ["8080:8080"]
			// dialog/asterisk в host network → host-gateway alias
			extra_hosts: ["asterisk:host-gateway", "dialog:host-gateway"]
			environment: {
				TRAINEEBOX_HTTP_ADDR:    ":8080"
				TRAINEEBOX_POSTGRES_DSN: "postgres://\(config.postgres.user):\(config.postgres.password)@postgres:5432/\(config.postgres.database)?sslmode=disable"
				TRAINEEBOX_CATALOG_PATH: "/catalog"
				INTERNAL_SERVICE_TOKEN:  config.service_token
				AUDIO_URL:              config.audio.traineebox_audio_url
				DIALOG_URL:              "http://dialog:8000"
				// AudioSocket в dialplan trainer-out ждёт именно dialog:9001
				// (Asterisk видит dialog через extra_hosts: dialog:host-gateway).
				DIALOG_ADDR:             "dialog:9001"
				ARI_URL:                 "http://asterisk:8088/ari"
				ARI_USER:                config.ari.user
				ARI_PASS:                config.ari.password
			}
			volumes: [
				"\(_catalog):/catalog:ro",
			]
			restart: "unless-stopped"
		}
		dialog: {
			build: {
				context:    "."
				dockerfile: "dialog/Dockerfile.prod"
			}
			gpus: "all"
			// host network: Asterisk (тоже host) видит dialog на 127.0.0.1:9001
			// без DNS, traineebox — через host-gateway alias. Без этого ARI
			// originate прокидывает "dialog:9001" в channel vars, и AudioSocket
			// падает с "Temporary failure in name resolution" — звонок сбрасывается.
			network_mode: "host"
			environment: {
				DIALOG_PORT:            "8000"
				DIALOG_AUDIOSOCKET:     "0.0.0.0:9001"
				DIALOG_DEVICE:          config.dialog.device
				DIALOG_PROD:            "1"
				AUDIO_URL:              config.audio.dialog_audio_url
				AUDIO_CACHE_DIR:         "/tmp/dialog-audio"
				AUDIO_CACHE_MB:          "500"
				// llama слушает на хосте (ports 8081:8081), dialog в host net
				// видит его на 127.0.0.1:8081.
				DISPATCHER_LLM_URL:     "http://127.0.0.1:8081"
				GIGAAM_MODEL_PATH:      "/app/models/gigaam-v2-ctc"
				DISPATCHER_MODELS:      "/app/dialog/models"
				// torch hub (Silero) кэш на volume, иначе каждый recreate
				// перекачивает silero с github (SSL/таймауты рвут boot).
				TORCH_HOME:             "/app/dialog/models/torch"
				// HF-кэш (Laya) на volume. xet отключён: иначе 614MB модель
				// материализуется вне volume и перекачивается каждый load.
				HF_HOME:                "/app/dialog/models"
				HF_HUB_DISABLE_XET:     "1"
				TRAINEEBOX_URL:         "http://traineebox:8080"
				INTERNAL_SERVICE_TOKEN: config.service_token
			}
			volumes: [
				"./artifacts/models/gigaam-v2-ctc:/app/models/gigaam-v2-ctc:ro",
				"dialog_models:/app/dialog/models",
			]
			restart: "unless-stopped"
		}
		asterisk: {
			image: "andrius/asterisk:20"
			// host network: postgres доступен через localhost:5432
			// (compose postgres порт 5432:5432 проброшен на хост).
			// RTP через NAT докера ломает тайминги — host net стабильнее.
			network_mode: "host"
			// dialog тоже в host network → "dialog" через host-gateway = IP хоста.
			// dialplan AudioSocket(${call_id},${DIALOG_ADDR}) использует "dialog:9001"
			// как имя сервиса, без этого getaddrinfo падает.
			extra_hosts: ["dialog:host-gateway"]
			volumes: [
				"./config/asterisk/pjsip.conf:/etc/asterisk/pjsip.conf:ro",
				"./config/asterisk/extensions.conf:/etc/asterisk/extensions.conf:ro",
				"./config/asterisk/http.conf:/etc/asterisk/http.conf:ro",
				"./config/asterisk/ari.conf:/etc/asterisk/ari.conf:ro",
				"./config/asterisk/extconfig.conf:/etc/asterisk/extconfig.conf:ro",
				"./config/asterisk/sorcery.conf:/etc/asterisk/sorcery.conf:ro",
				"./config/asterisk/res_pgsql.conf:/etc/asterisk/res_pgsql.conf:ro",
			]
			restart: "unless-stopped"
		}
		"audio-postgres": {
			image: "postgres:16"
			environment: {
				POSTGRES_USER:     config.audio.postgres.user
				POSTGRES_PASSWORD: config.audio.postgres.password
				POSTGRES_DB:       config.audio.postgres.database
			}
			ports: ["5433:5432"]
			volumes: [
				"audio_postgres_data:/var/lib/postgresql/data",
			]
			healthcheck: {
				test: ["CMD-SHELL", "pg_isready -U \(config.audio.postgres.user)"]
				interval: "5s"
				timeout:  "3s"
				retries:   5
			}
			restart: "unless-stopped"
		}
		"audio-migrate": {
			build: {
				context:    "."
				dockerfile: "audio/Dockerfile"
			}
			command: ["/app/audio", "migrate", "up"]
			depends_on: {
				"audio-postgres": {
					condition: "service_healthy"
				}
			}
			environment: {
				AUDIO_DATABASE_URL: _audioDatabaseURL
			}
			restart: "no"
		}
		audio: {
			build: {
				context:    "."
				dockerfile: "audio/Dockerfile"
			}
			depends_on: {
				"audio-migrate": {
					condition: "service_completed_successfully"
				}
			}
			ports: ["8002:8002"]
			environment: {
				AUDIO_HTTP_ADDR:       ":8002"
				AUDIO_DATABASE_URL:    _audioDatabaseURL
				AUDIO_S3_ENDPOINT:     "http://rustfs:9000"
				AUDIO_S3_BUCKET:       "emergency-audio"
				AUDIO_S3_KEY:          config.audio.s3.access_key
				AUDIO_S3_SECRET:       config.audio.s3.secret_key
				AUDIO_VOICE:           config.audio.voice
				AUDIO_RATE:            "\(config.audio.rate)"
				AUDIO_SYNTH_URL:       "http://audio-synth:8003"
				AUDIO_SWEEP_MINUTES:   "\(config.audio.sweep_minutes)m"
				TRAINEEBOX_URL:         config.audio.traineebox_url
				INTERNAL_SERVICE_TOKEN: config.service_token
			}
			restart: "unless-stopped"
		}
		"audio-synth": {
			build: {
				context:    "."
				dockerfile: "audio-synth/Dockerfile"
			}
			gpus: "all"
			environment: {
				SYNTH_PORT:   "8003"
				SYNTH_DEVICE: config.audio.synth_device
				TORCH_HOME:   "/models/torch"
				HF_HOME:      "/models"
			}
			volumes: [
				"audio_models:/models",
			]
			restart: "unless-stopped"
		}
		rustfs: {
			image: "rustfs/rustfs:latest"
			ports: ["9002:9000", "9003:9001"]
			environment: {
				RUSTFS_VOLUMES:               "/data/rustfs0"
				RUSTFS_ADDRESS:               "0.0.0.0:9000"
				RUSTFS_CONSOLE_ADDRESS:       "0.0.0.0:9001"
				RUSTFS_CONSOLE_ENABLE:        "true"
				RUSTFS_ACCESS_KEY:            config.audio.s3.access_key
				RUSTFS_SECRET_KEY:            config.audio.s3.secret_key
				RUSTFS_OBS_LOG_DIRECTORY:     "/app/logs"
				RUSTFS_UNSAFE_BYPASS_DISK_CHECK: "true"
			}
			volumes: [
				"rustfs_data:/data/rustfs0",
				"rustfs_logs:/app/logs",
			]
			restart: "unless-stopped"
		}
	}
	volumes: {
		postgres_data: {}
		caddy_data:    {}
		caddy_config:  {}
		dialog_models: {}
		audio_postgres_data: {}
		audio_models: {}
		rustfs_data: {}
		rustfs_logs: {}
	}
}
