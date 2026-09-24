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
}

config: #Config

_caddyfile: "\(strings.TrimSuffix(config.root, "/"))/etc/caddy/Caddyfile"
_catalog:   "\(strings.TrimSuffix(config.root, "/"))/etc/traineebox/catalog"

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
		ticketgen: {
			build: {
				context:    "."
				dockerfile: "ticketgen/Dockerfile"
			}
			depends_on: ["postgres"]
			environment: {
				TICKETGEN_POSTGRES_DSN:  "postgres://\(config.postgres.user):\(config.postgres.password)@postgres:5432/\(config.postgres.database)?sslmode=disable"
				TICKETGEN_CATALOG_PATH:  "/catalog"
				TICKETGEN_LLM_BACKENDS:  _llmBackendsEnv
				TICKETGEN_POLL_SECONDS:  "\(config.ticketgen.poll_seconds)"
				TICKETGEN_LEASE_SECONDS: "\(config.ticketgen.lease_seconds)"
			}
			volumes: [
				"\(_catalog):/catalog:ro",
			]
			restart: "unless-stopped"
		}
	}
	volumes: {
		postgres_data: {}
		caddy_data:    {}
		caddy_config:  {}
	}
}
