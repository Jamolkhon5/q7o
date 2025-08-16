# 1. Создать сеть Docker
make network-create

# 2. Запустить LiveKit
make livekit-up

# 3. Запустить приложение с БД и Redis
make docker-up

# Или всё вместе
make up

# Остановить всё
make down

# Логи
make logs          # логи приложения
make livekit-logs  # логи LiveKit