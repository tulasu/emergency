package main

config: {
	root: "./artifacts"
	postgres: {
		user:     "emergency"
		password: "emergency"
		database: "emergency"
	}
	ticketgen: {
		llm: {
			backends: [
				{
					name:     "qwen3-1.7b"
					url:      "http://host.docker.internal:8081"
					model:    "Qwen3-1.7B-Q8_0.gguf"
					priority: 1
					timeout:  120
				},
			]
		}
	}
	// dev defaults: prod overrides token + ari/admin passwords (ari.conf [admin] must match ari.password)
	service_token: "dev-service-token-change-me"
	ari: {
		user:     "admin"
		password: "admin"
	}
	admin: {
		login:    "admin"
		password: "adminadmin"
	}
	dialog: {
		device: "cpu"
	}
}
