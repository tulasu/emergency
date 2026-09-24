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
					model:    "Qwen3-1.7B-Q4_K_M.gguf"
					priority: 1
					timeout:  120
				},
			]
		}
	}
}
