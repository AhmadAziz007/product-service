package message

import "github.com/labstack/gommon/log"

// StartUpdateStockConsumer is kept as a compatibility command. Stock is now
// reserved synchronously by the inventory repository using an atomic update.
func StartUpdateStockConsumer() {
	log.Warn("stock RabbitMQ consumer is disabled; inventory reservations are handled by product-service")
}
