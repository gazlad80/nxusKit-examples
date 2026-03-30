// Riffer CLI - Music Sequence Analysis and Transformation Tool
//
// A command-line tool for analyzing and transforming music sequences.
//
// ## Interactive Modes
// - --verbose or -v: Shows raw request/response data for debugging
// - --step or -s: Pauses at each major operation with explanations
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nxus-SYSTEMS/nxusKit/examples/apps/riffer"
	"github.com/nxus-SYSTEMS/nxusKit/examples/apps/riffer/llm"
	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
)

// Global interactive config
var interactiveConfig *interactive.Config

const version = "0.1.0"

func main() {
	// Parse interactive mode flags first
	interactiveConfig = interactive.FromArgs()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "analyze":
		runAnalyze(os.Args[2:])
	case "score":
		runScore(os.Args[2:])
	case "convert":
		runConvert(os.Args[2:])
	case "transform":
		runTransform(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("riffer version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Riffer - Music sequence analysis and transformation tool

Usage:
  riffer <command> [options]

Commands:
  analyze    Analyze a music sequence
  score      Score a music sequence
  convert    Convert between formats
  transform  Transform a music sequence
  version    Print version information
  help       Print this help message

Interactive Modes:
  -v, --verbose    Show raw request/response data for debugging
  -s, --step       Step through operations with pauses

Run 'riffer <command> --help' for more information on a command.`)
}

func runAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	input := fs.String("input", "", "Input file (MIDI or MusicXML)")
	inputShort := fs.String("i", "", "Input file (short)")
	format := fs.String("format", "json", "Output format (json, markdown)")
	formatShort := fs.String("f", "", "Output format (short)")
	narrative := fs.Bool("narrative", false, "Use LLM to generate narrative analysis")
	fs.Parse(args)

	// Handle short flags
	if *inputShort != "" && *input == "" {
		*input = *inputShort
	}
	if *formatShort != "" && *format == "json" {
		*format = *formatShort
	}

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Error: input file required (-i or --input)")
		fs.Usage()
		os.Exit(1)
	}

	// Step: Reading MIDI file
	if action := interactiveConfig.StepPause("Reading MIDI file...", []string{
		fmt.Sprintf("Input: %s", *input),
		"Parsing MIDI events and note data",
	}); action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Read sequence
	seq, err := riffer.ReadMIDI(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Set name from filename if not set
	if seq.Name == nil {
		base := filepath.Base(*input)
		name := base[:len(base)-len(filepath.Ext(base))]
		seq.Name = &name
	}

	// Step: Analyzing sequence
	if action := interactiveConfig.StepPause("Analyzing music sequence...", []string{
		"Detecting key signature",
		"Analyzing intervals and scales",
		"Evaluating melodic contour",
		"Examining rhythm patterns",
	}); action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Analyze
	analysis, err := riffer.AnalyzeSequence(seq, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing: %v\n", err)
		os.Exit(1)
	}

	// Output
	if *format == "markdown" {
		printMarkdownAnalysis(analysis)
	} else {
		data, _ := json.MarshalIndent(analysis, "", "  ")
		fmt.Println(string(data))
	}

	// Generate narrative if requested
	if *narrative {
		// Step: Generating LLM narrative
		if action := interactiveConfig.StepPause("Generating LLM narrative analysis...", []string{
			"Sending analysis data to LLM",
			"LLM will generate human-readable interpretation",
		}); action == interactive.ActionQuit {
			fmt.Println("Exiting...")
			return
		}

		fmt.Println("\n## Narrative Analysis")
		ctx := context.Background()
		narrativeText, err := llm.GenerateNarrative(ctx, analysis)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not generate narrative: %v\n", err)
		} else {
			fmt.Println(narrativeText)
		}
	}
}

func printMarkdownAnalysis(analysis *riffer.AnalysisResult) {
	name := "Unknown"
	if analysis.Name != "" {
		name = analysis.Name
	}

	fmt.Printf("# Music Analysis: %s\n\n", name)

	fmt.Println("## Summary")
	fmt.Printf("- **Notes**: %d\n", analysis.NoteCount)
	fmt.Printf("- **Duration**: %d ticks\n", analysis.DurationTicks)

	fmt.Println("\n## Key Detection")
	fmt.Printf("- **Detected Key**: %s\n", analysis.KeyDetection.Key.String())
	fmt.Printf("- **Confidence**: %.1f%%\n", analysis.KeyDetection.Confidence*100.0)
	if len(analysis.KeyDetection.Alternatives) > 0 {
		fmt.Println("- **Alternatives**:")
		for i, alt := range analysis.KeyDetection.Alternatives {
			if i >= 3 {
				break
			}
			fmt.Printf("  - %s (%.2f)\n", alt.Key.String(), alt.Correlation)
		}
	}

	fmt.Println("\n## Scale Analysis")
	fmt.Printf("- **Key**: %s\n", analysis.ScaleAnalysis.Key.String())
	fmt.Printf("- **In Scale**: %d notes\n", analysis.ScaleAnalysis.InScaleCount)
	fmt.Printf("- **Out of Scale**: %d notes\n", analysis.ScaleAnalysis.OutOfScaleCount)
	fmt.Printf("- **Harmonic Coherence**: %.1f%%\n", analysis.ScaleAnalysis.CoherencePercentage)

	fmt.Println("\n## Interval Analysis")
	fmt.Printf("- **Total Intervals**: %d\n", analysis.IntervalAnalysis.Count)
	fmt.Printf("- Perfect Consonances: %d\n", analysis.IntervalAnalysis.ByQuality.PerfectConsonance)
	fmt.Printf("- Imperfect Consonances: %d\n", analysis.IntervalAnalysis.ByQuality.ImperfectConsonance)
	fmt.Printf("- Mild Dissonances: %d\n", analysis.IntervalAnalysis.ByQuality.MildDissonance)
	fmt.Printf("- Strong Dissonances: %d\n", analysis.IntervalAnalysis.ByQuality.StrongDissonance)
	fmt.Printf("- **Interval Variety**: %d unique intervals\n", analysis.IntervalAnalysis.IntervalVariety)

	fmt.Println("\n## Melodic Contour")
	fmt.Printf("- **Contour Type**: %s\n", analysis.ContourAnalysis.ContourType.String())
	fmt.Printf("- **Direction Changes**: %d\n", analysis.ContourAnalysis.DirectionChanges)
	fmt.Printf("- **Pitch Range**: %d semitones (%d to %d)\n",
		analysis.ContourAnalysis.PitchRange,
		analysis.ContourAnalysis.LowestPitch,
		analysis.ContourAnalysis.HighestPitch)

	fmt.Println("\n## Rhythm Analysis")
	fmt.Printf("- **Unique Durations**: %d\n", analysis.RhythmAnalysis.UniqueDurations)
	fmt.Printf("- **Most Common Duration**: %d ticks\n", analysis.RhythmAnalysis.MostCommonDuration)
	fmt.Printf("- **Duration Variety**: %.2f\n", analysis.RhythmAnalysis.DurationVariety)

	fmt.Println("\n## Dynamics Analysis")
	fmt.Printf("- **Velocity Range**: %d - %d\n",
		analysis.DynamicsAnalysis.MinVelocity,
		analysis.DynamicsAnalysis.MaxVelocity)
	hasDyn := "No"
	if analysis.DynamicsAnalysis.HasDynamics {
		hasDyn = "Yes"
	}
	fmt.Printf("- **Has Dynamics**: %s\n", hasDyn)
}

func runScore(args []string) {
	fs := flag.NewFlagSet("score", flag.ExitOnError)
	input := fs.String("input", "", "Input file (MIDI or MusicXML)")
	inputShort := fs.String("i", "", "Input file (short)")
	format := fs.String("format", "json", "Output format (json, markdown)")
	formatShort := fs.String("f", "", "Output format (short)")
	useClips := fs.Bool("clips", false, "Enable CLIPS rule engine for scoring adjustments")
	rulesDir := fs.String("rules-dir", "", "Path to CLIPS rules directory (default: ./rules)")
	fs.Parse(args)

	// Handle short flags
	if *inputShort != "" && *input == "" {
		*input = *inputShort
	}
	if *formatShort != "" && *format == "json" {
		*format = *formatShort
	}

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Error: input file required (-i or --input)")
		fs.Usage()
		os.Exit(1)
	}

	// Read sequence
	seq, err := riffer.ReadMIDI(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Set name from filename if not set
	if seq.Name == nil {
		base := filepath.Base(*input)
		name := base[:len(base)-len(filepath.Ext(base))]
		seq.Name = &name
	}

	// Determine rules directory
	effectiveRulesDir := ""
	if *useClips {
		if *rulesDir != "" {
			effectiveRulesDir = *rulesDir
		} else {
			effectiveRulesDir = riffer.LoadRulesDir()
		}
	}

	// Score with optional CLIPS
	var score *riffer.MusicScore
	if effectiveRulesDir != "" {
		ctx := context.Background()
		score, err = riffer.ScoreSequenceWithClips(ctx, seq, effectiveRulesDir)
	} else {
		score, err = riffer.ScoreSequence(seq, false)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scoring: %v\n", err)
		os.Exit(1)
	}

	// Output
	if *format == "markdown" {
		printMarkdownScore(seq.Name, score)
	} else {
		data, _ := json.MarshalIndent(score, "", "  ")
		fmt.Println(string(data))
	}
}

func printMarkdownScore(name *string, score *riffer.MusicScore) {
	seqName := "Unknown"
	if name != nil {
		seqName = *name
	}

	fmt.Printf("# Music Score: %s\n\n", seqName)

	fmt.Printf("## Overall Score: %.1f/100 (%s)\n\n", score.Overall, score.Summary.Rating)

	fmt.Println("## Dimension Scores")
	fmt.Println("| Dimension | Score | Rating |")
	fmt.Println("|-----------|-------|--------|")
	fmt.Printf("| Harmonic Coherence | %.1f | %s |\n",
		score.Dimensions.HarmonicCoherence.Score,
		score.Dimensions.HarmonicCoherence.Rating)
	fmt.Printf("| Melodic Interest | %.1f | %s |\n",
		score.Dimensions.MelodicInterest.Score,
		score.Dimensions.MelodicInterest.Rating)
	fmt.Printf("| Rhythmic Variety | %.1f | %s |\n",
		score.Dimensions.RhythmicVariety.Score,
		score.Dimensions.RhythmicVariety.Rating)
	fmt.Printf("| Resolution Quality | %.1f | %s |\n",
		score.Dimensions.ResolutionQuality.Score,
		score.Dimensions.ResolutionQuality.Rating)
	fmt.Printf("| Dynamics Expression | %.1f | %s |\n",
		score.Dimensions.DynamicsExpression.Score,
		score.Dimensions.DynamicsExpression.Rating)
	fmt.Printf("| Structural Balance | %.1f | %s |\n",
		score.Dimensions.StructuralBalance.Score,
		score.Dimensions.StructuralBalance.Rating)

	fmt.Println("\n## Summary")
	fmt.Printf("- **Strongest**: %s\n", score.Summary.Strongest)
	fmt.Printf("- **Weakest**: %s\n", score.Summary.Weakest)
	fmt.Printf("\n%s\n", score.Summary.Summary)

	if len(score.Suggestions) > 0 {
		fmt.Println("\n## Suggestions for Improvement")
		for _, suggestion := range score.Suggestions {
			fmt.Printf("- %s\n", suggestion)
		}
	}
}

func runConvert(args []string) {
	fs := flag.NewFlagSet("convert", flag.ExitOnError)
	input := fs.String("input", "", "Input file (MIDI or MusicXML)")
	inputShort := fs.String("i", "", "Input file (short)")
	output := fs.String("output", "", "Output file")
	outputShort := fs.String("o", "", "Output file (short)")
	fs.Parse(args)

	// Handle short flags
	if *inputShort != "" && *input == "" {
		*input = *inputShort
	}
	if *outputShort != "" && *output == "" {
		*output = *outputShort
	}

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "Error: input and output files required")
		fs.Usage()
		os.Exit(1)
	}

	// Detect input format and read
	inputExt := filepath.Ext(*input)
	var seq *riffer.Sequence
	var err error

	switch inputExt {
	case ".mid", ".midi":
		seq, err = riffer.ReadMIDI(*input)
	case ".xml", ".musicxml", ".mxl":
		seq, err = riffer.ReadMusicXML(*input)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported input format '%s' (use .mid, .midi, .xml, or .musicxml)\n", inputExt)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Detect output format and write
	outputExt := filepath.Ext(*output)
	switch outputExt {
	case ".mid", ".midi":
		err = riffer.WriteMIDI(seq, *output)
	case ".xml", ".musicxml", ".mxl":
		err = riffer.WriteMusicXML(seq, *output)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported output format '%s' (use .mid, .midi, .xml, or .musicxml)\n", outputExt)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Converted %s to %s\n", *input, *output)
}

func runTransform(args []string) {
	fs := flag.NewFlagSet("transform", flag.ExitOnError)
	input := fs.String("input", "", "Input file (MIDI or MusicXML)")
	inputShort := fs.String("i", "", "Input file (short)")
	output := fs.String("output", "", "Output file")
	outputShort := fs.String("o", "", "Output file (short)")
	transpose := fs.Int("transpose", 0, "Transpose by semitones (-12 to +12)")
	tempo := fs.Int("tempo", 0, "Change tempo (BPM, 20-300)")
	invertFlag := fs.Bool("invert", false, "Invert melody around first note")
	invertPivot := fs.Int("invert-pivot", -1, "Invert melody around specific pitch (0-127)")
	retrograde := fs.Bool("retrograde", false, "Reverse note order")
	augmentFactor := fs.Float64("augment", 0, "Augment durations by factor (0.125-8.0)")
	diminishFactor := fs.Float64("diminish", 0, "Diminish durations by factor (0.125-8.0)")
	keyChange := fs.String("key", "", "Change key (e.g., 'C', 'Am', 'F#m')")
	prompt := fs.String("prompt", "", "Natural language transformation prompt (requires LLM)")
	fs.Parse(args)

	// Handle short flags
	if *inputShort != "" && *input == "" {
		*input = *inputShort
	}
	if *outputShort != "" && *output == "" {
		*output = *outputShort
	}

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "Error: input and output files required")
		fs.Usage()
		os.Exit(1)
	}

	// Read
	seq, err := riffer.ReadMIDI(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Apply LLM-interpreted transformation if prompt is provided
	if *prompt != "" {
		ctx := context.Background()
		description, err := llm.TransformWithPrompt(ctx, seq, *prompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error applying LLM transformation: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "LLM transformation: %s\n", description)
	}

	// Apply transpose
	if *transpose != 0 {
		if err := riffer.Transpose(seq, int8(*transpose)); err != nil {
			fmt.Fprintf(os.Stderr, "Error transposing: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Transposed by %d semitones\n", *transpose)
	}

	// Apply tempo change
	if *tempo > 0 {
		if err := riffer.ChangeTempo(seq, uint16(*tempo)); err != nil {
			fmt.Fprintf(os.Stderr, "Error changing tempo: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Set tempo to %d BPM\n", *tempo)
	}

	// Apply invert
	if *invertFlag || *invertPivot >= 0 {
		var pivot *uint8
		if *invertPivot >= 0 {
			p := uint8(*invertPivot)
			pivot = &p
		}
		if err := riffer.Invert(seq, pivot); err != nil {
			fmt.Fprintf(os.Stderr, "Error inverting: %v\n", err)
			os.Exit(1)
		}
		if pivot != nil {
			fmt.Fprintf(os.Stderr, "Inverted around pitch %d\n", *pivot)
		} else {
			fmt.Fprintln(os.Stderr, "Inverted around first note")
		}
	}

	// Apply retrograde
	if *retrograde {
		if err := riffer.Retrograde(seq); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying retrograde: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Applied retrograde (reversed notes)")
	}

	// Apply augment
	if *augmentFactor > 0 {
		if err := riffer.Augment(seq, float32(*augmentFactor)); err != nil {
			fmt.Fprintf(os.Stderr, "Error augmenting: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Augmented durations by factor %v\n", *augmentFactor)
	}

	// Apply diminish
	if *diminishFactor > 0 {
		if err := riffer.Diminish(seq, float32(*diminishFactor)); err != nil {
			fmt.Fprintf(os.Stderr, "Error diminishing: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Diminished durations by factor %v\n", *diminishFactor)
	}

	// Apply key change
	if *keyChange != "" {
		targetKey, ok := riffer.ParseKey(*keyChange)
		if !ok {
			fmt.Fprintf(os.Stderr, "Invalid key format: '%s'. Use format like 'C', 'Am', 'F#m'\n", *keyChange)
			os.Exit(1)
		}
		if err := riffer.KeyChange(seq, *targetKey); err != nil {
			fmt.Fprintf(os.Stderr, "Error changing key: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Changed key to %s\n", targetKey.String())
	}

	// Write
	if err := riffer.WriteMIDI(seq, *output); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Written to %s\n", *output)
}
