package main

import "strings"

#Config: {
	root: *"/" | string

	postgres: {
		user:     string & =~"^.+$"
		password: string & =~"^.+$"
		database: string & =~"^.+$"
	}
}

config: #Config

_caddyfile: "\(strings.TrimSuffix(config.root, "/"))/etc/caddy/Caddyfile"

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
	}
	volumes: {
		postgres_data: {}
		caddy_data:    {}
		caddy_config:  {}
	}
}
