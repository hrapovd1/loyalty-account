# Накопительная система лояльности «Гофермарт»

**Дипломный проект курса «Go-разработчик».**

### Общее описание задания.

Система представляет собой HTTP API со следующими требованиями к бизнес-логике:
   * регистрация, аутентификация и авторизация пользователей;
   * приём номеров заказов от зарегистрированных пользователей;
   * учёт и ведение списка переданных номеров заказов зарегистрированного пользователя;
   * учёт и ведение накопительного счёта зарегистрированного пользователя;
   * проверка принятых номеров заказов через систему расчёта баллов лояльности;
   * начисление за каждый подходящий номер заказа положенного вознаграждения на счёт лояльности пользователя.

**Сводное HTTP API**

Накопительная система лояльности «Гофермарт» должна предоставлять следующие HTTP-хендлеры:

      POST /api/user/register — регистрация пользователя;

      POST /api/user/login — аутентификация пользователя;

      POST /api/user/orders — загрузка пользователем номера заказа для расчёта;

      GET /api/user/orders — получение списка загруженных пользователем номеров заказов, статусов их обработки и информации о начислениях;

      GET /api/user/balance — получение текущего баланса счёта баллов лояльности пользователя;

      POST /api/user/balance/withdraw — запрос на списание баллов с накопительного счёта в счёт оплаты нового заказа;

      GET /api/user/withdrawals — получение информации о выводе средств с накопительного счёта пользователем.

## Сборка и запуск 

```BASH
go buil -o app cmd/gophermart/main.go
```
Для работы приложения необходима БД postgresql > 13 и доступ к 
системе начисления баллов.

Опции запуска приложения можно получить при запуске с ключем -h

```BASH
./app -h
Usage of ./app:
  -a string
        Address of application, for example: 0.0.0.0:8000
  -d string
        Database connect source, for example: postgres://username:password@localhost:5432/database_name
  -r string
        Accrual system address, for example: localhost:8080
```

В результате приложение запуститься и создаст необходимые таблицы в базе данных.
