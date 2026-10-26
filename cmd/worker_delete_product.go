package cmd

import (
	"ecommerce/product-service/internal/adapter/message"
	"fmt"

	"github.com/spf13/cobra"
)

var workerDeleteCmd = &cobra.Command{
	Use:   "worker-delete-product",
	Short: "Menjalankan worker untuk consume RabbitMQ dan index ke Elasticsearch",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Worker untuk Elasticsearch Indexing sedang berjalan...")
		message.StartDeleteOrderConsumer()
	},
}

func init() {
	rootCmd.AddCommand(workerDeleteCmd)
}
