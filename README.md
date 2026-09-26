# Тренажёр вызовов службы **112**

![Angular](https://img.shields.io/badge/Angular-21-DD0031?style=flat-square&logo=angular&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?style=flat-square&logo=typescript&logoColor=white)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)
![Python](https://img.shields.io/badge/Python-3.11+-3776AB?style=flat-square&logo=python&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat-square&logo=postgresql&logoColor=white)
![Asterisk](https://img.shields.io/badge/Asterisk-20-FBB040?style=flat-square&logoColor=black)
![Caddy](https://img.shields.io/badge/Caddy-2-22D3EE?style=flat-square&logo=caddy&logoColor=black)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker&logoColor=white)
![CUE](https://img.shields.io/badge/CUE-config-E85D04?style=flat-square)
![llama.cpp](https://img.shields.io/badge/llama.cpp-Qwen-1B1B1B?style=flat-square)

## Где что лежит

- [`web/`](web/) — клиент на Angular: вход, кабинет, пользователи и группы
- [`traineebox/`](traineebox/) — backend на Go: API, сессии, билеты, попытки, старт звонка через ARI
- [`dialog/`](dialog/) — голосовой движок на Python: STT/TTS, NLU, сценарий заявителя по AudioSocket
- [`ticketgen/`](ticketgen/) — AI-пайплайн на Python: черновики учебных сценариев из каталога и LLM

## Лицензия

Разработано в рамках хакатона © 2026 Команда «Токеноежки»
