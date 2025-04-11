package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/shammianand/picalc/pkg/picalc"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:     "picalc",
		Short:   "A blazingly fast π calculator",
		Long:    `Calculates π to a specified number of decimal digits using the Chudnovsky algorithm with maximum concurrency.`,
		Version: picalc.VERSION,
	}

	var calculateCmd = &cobra.Command{
		Use:   "calculate [digits]",
		Short: "Calculate π to the specified number of digits",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			digits, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				fmt.Println("Error: digits must be a valid integer")
				os.Exit(1)
			}

			outputFile, _ := cmd.Flags().GetString("output")
			showProgress, _ := cmd.Flags().GetBool("progress")

			calculatePi(digits, outputFile, showProgress)
		},
	}

	calculateCmd.Flags().StringP("output", "o", "", "Save digits to file")
	calculateCmd.Flags().BoolP("progress", "p", true, "Show progress bar")

	rootCmd.AddCommand(calculateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func calculatePi(digits int64, outputFile string, showProgress bool) {
	fmt.Printf("Calculating π to %d decimal digits...\n", digits)
	startTime := time.Now()

	// Create a progress tracker if enabled
	var bar *progressbar.ProgressBar
	var progressSignal chan struct{}

	if showProgress {
		bar = progressbar.DefaultBytes(
			digits,
			"Computing",
		)
		progressSignal = make(chan struct{})
	}

	// Start Pi calculation in a goroutine
	pi := picalc.NewPi(digits)
	done := make(chan struct{})

	go func() {
		picalc.CalculatePi(digits, pi)
		close(done)
	}()

	// Update progress if enabled
	if showProgress {
		go func() {
			for {
				select {
				case <-progressSignal:
					return
				default:
					progress := pi.GetProgress()
					bar.Set64(int64(float64(digits) * progress / 100.0))
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
	}

	// Wait for completion
	<-done
	if showProgress {
		close(progressSignal)
		bar.Finish()
	}

	// Get all digits
	piDigits := pi.GetDigits(int(digits))

	// Calculate elapsed time
	duration := time.Since(startTime)
	fmt.Printf("\nCalculation completed in %v\n", duration)

	// Output results
	if outputFile != "" {
		picalc.WriteDigitsToFile(piDigits, outputFile)
		fmt.Printf("Results saved to %s\n", outputFile)
	} else {
		fmt.Print("π = 3.")
		for i := 1; i < len(piDigits) && i <= 100; i++ {
			fmt.Print(piDigits[i])
		}
		if len(piDigits) > 100 {
			fmt.Print("...")
		}
		fmt.Println()
		fmt.Println("Use --output flag to save all digits to a file")
	}
}
