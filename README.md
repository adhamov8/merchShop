# Avito Merch Shop

Создал сервис, который позволит сотрудникам обмениваться монетками и приобретать на них мерч.
Каждый новый пользователь при регистрации получает 1000 монет, может покупать на них мерч и передавать монеты другим сотрудникам.

## Описание

Проект решает задачу из условия:
- При первом **логине/регистрации** вы создаёте пользователя с **1000 монет**.
- **Авторизация** организована через **JWT**: пользователь получает токен, которым должен подписывать запросы к защищённым эндпоинтам.
- **Покупка мерча** доступна при наличии достаточного количества монет (запрещено уходить в минус).
- **Передача монет** между пользователями — аналогично, без ухода в минус.
- **История** транзакций (кто кому отправлял и сколько) и купленных предметов возвращается через эндпоинт `/api/info`.

## Установка и запуск

### Запуск с помощью Docker Compose

1. Убедитесь, что у вас установлены Docker и Docker Compose.
2. Склонируйте репозиторий (или скачайте файлы).
3. Выполните в терминале:
   ```bash
   docker-compose up --build
   ```
4. После успешной сборки сервис будет доступен по адресу `http://localhost:8080`.
5. По умолчанию база Postgres будет мапиться на порт `5432` хоста. Если у вас локально уже запущен Postgres на 5432

> **Важно:** Если ваш локальный порт 5432 занят, произойдёт конфликт. Либо остановите локальный Postgres, либо смените порт.

### Настройки переменных окружения

По умолчанию, в `docker-compose.yml` указаны следующие переменные окружения:
- DATABASE_HOST, DATABASE_PORT, DATABASE_USER, DATABASE_PASSWORD, DATABASE_NAME - настройки БД
- SERVER_PORT - HTTP порт
- JWT_SECRET - секретный ключ JWT

 можно изменять `.env` или напрямую править `docker-compose.yml`.

---

## Использование

По умолчанию, если открыть `http://localhost:8080/`, вы получите простую HTML-страницу, которая перечисляет основные эндпоинты:

### 1. Регистрация / Аутентификация (`POST /api/auth`)

**Параметры (JSON в теле):**
```json
{
  "username": "TestUser",
  "password": "Valid@Pass123"
}
```
- Если пользователь не существует, он создаётся (с балансом 1000 монет).
- Если существует, проверяется пароль. При ошибке вернётся `401 {"errors":"invalid credentials"}`.
- При слабом пароле — `400 {"errors":"weak password"}`.

**Успешный ответ (JSON):**
```json
{
  "token": "<JWT>"
}
```

Сохраните значение `token`, чтобы использовать для защищённых эндпоинтов.

### 2. Получение информации (`GET /api/info`)

- **Защищённый** эндпоинт: требуются заголовок `Authorization: Bearer <token>`.
- Возвращает баланс, инвентарь (список {тип предмета, количество}), а также историю транзакций (кто отправлял, кому отправляли).

Пример ответа:
```json
{
  "coins": 950,
  "inventory": [
    {
      "type": "book",
      "quantity": 1
    }
  ],
  "coinHistory": {
    "received": [],
    "sent": []
  }
}
```

### 3. Отправка монет (`POST /api/sendCoin`)

- **Защищённый** эндпоинт.
- Тело (JSON):
  ```json
  {
    "toUser": "Alibek",
    "amount": 100
  }
  ```
- Успешный ответ:
  ```json
  {
    "status": "ok"
  }
  ```
- Если монет недостаточно или пользователь не найден, будет `400`.

### 4. Покупка мерча (`GET /api/buy/{item}`)

- **Защищённый** эндпоинт.
- `{item}` может быть: `t-shirt`, `cup`, `book`, `pen`, `powerbank`, `hoody`, `umbrella`, `socks`, `wallet`, `pink-hoody`.
- Пример: `GET /api/buy/book`.
- При успехе вернётся:
  ```json
  {
    "status": "ok"
  }
  ```
- Если монет недостаточно — `400 {"errors":"not enough coins"}`.

### Пример

1. **Регистрация**:
   ```bash
   curl -X POST "http://localhost:8080/api/auth" \
        -H "Content-Type: application/json" \
        -d '{"username":"Ziyo","password":"Valid@Pass123"}'
   ```
   Ответ:
   ```json
   {"token": "eyJhbGciOiJIUz..." }
   ```

2. **GET /api/info**:
   ```bash
   curl -X GET "http://localhost:8080/api/info" \
        -H "Authorization: Bearer eyJhbGciOiJIUz..."
   ```
   Ответ (например):
   ```json
   {"coins":1000,"inventory":[],"coinHistory":{"received":[],"sent":[]}}
   ```

3. **Покупка**:
   ```bash
   curl -X GET "http://localhost:8080/api/buy/book" \
        -H "Authorization: Bearer eyJhbGciOiJIUz..."
   ```
   Ответ:
   ```json
   {"status":"ok"}
   ```

---

## Покрытие тестами

Чтобы **запустить тесты** локально (не через Docker):
```bash
go test -cover ./...
```
---

## Другое

> Скриншот работы программы, а также результаты тестов прикреплины в папке screenshots

