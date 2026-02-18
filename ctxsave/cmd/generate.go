package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"ctxsave/internal/generate"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var (
	genModel  string
	genBudget int
	genCopy   bool
	genOut    string
)

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVar(&genModel, "model", "sonnet", "target model key (gemini, opus, sonnet, gpt4o)")
	generateCmd.Flags().IntVar(&genBudget, "budget", 0, "token budget (0 = auto based on model)")
	generateCmd.Flags().BoolVar(&genCopy, "copy", false, "copy generated prompt to clipboard")
	generateCmd.Flags().StringVar(&genOut, "out", "", "write prompt to file")
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a context prompt for pasting into a new AI session",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, project, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()

		gen := generate.NewPromptGenerator(st, project)
		prompt, err := gen.Generate(generate.GenerateOptions{
			ModelKey: genModel,
			Budget:   genBudget,
		})
		if err != nil {
			return err
		}

		if genCopy {
			if err := clipboard.WriteAll(prompt); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not copy to clipboard: %v\n", err)
			} else {
				fmt.Println("Prompt copied to clipboard!")
			}
		}

		if genOut != "" {
			outPath := genOut
			if !filepath.IsAbs(outPath) {
				dir, _ := os.Getwd()
				outPath = filepath.Join(dir, outPath)
			}
			if err := os.WriteFile(outPath, []byte(prompt), 0644); err != nil {
				return fmt.Errorf("write file: %w", err)
			}
			fmt.Printf("Prompt written to %s\n", outPath)
		}

		if !genCopy && genOut == "" {
			fmt.Println(prompt)
		}

		return nil
	},
}
