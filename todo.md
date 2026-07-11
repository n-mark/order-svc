- реализовать рэпозиторий




BROKER_TYPE=RABBITMQ RABBIT_USERNAME=guest RABBIT_PASSWORD=guest RABBIT_HOST=localhost 
RABBIT_PORT=5672 RABBIT_PROFILE_QUEUE=micha-1 RABBIT_ORDER_QUEUE=micha-2 
RABBIT_BILLING_EXCHANGE=billing RABBIT_BILLING_ACCOUNT_CREATED_RK=billing.account_created 
RABBIT_BILLING_PAYMENT_SUCCESS_RK=billing.payment_success 
RABBIT_BILLING_PAYMENT_REQUIRED_RK=billing.payment_required  go run .


+ убрать миграцию
+ подключить роутер
+ проверить
+ инициализация очередей
- поправить <