package cmd

import (
	"ecommerce/product-service/internal/adapter/message"
	"fmt"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Menjalankan worker untuk consume RabbitMQ dan index ke Elasticsearch",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Worker untuk ElasticSearch indexing sedang berjalan...")
		message.StartConsumer()
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
}
