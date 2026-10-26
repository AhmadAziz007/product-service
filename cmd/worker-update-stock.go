package cmd

import (
	"ecommerce/product-service/internal/adapter/message"
	"fmt"

	"github.com/spf13/cobra"
)

var workerUpdateStockCmd = &cobra.Command{
	Use:   "worker-update-stock",
	Short: "Legacy stock worker (disabled)",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Stock is reserved synchronously by product-service; legacy worker is disabled.")
		message.StartUpdateStockConsumer()
	},
}

func init() {
	rootCmd.AddCommand(workerUpdateStockCmd)
}
