package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"

	"github.com/robotii/lito/compiler"
	"github.com/robotii/lito/compiler/parser"
	"github.com/robotii/lito/repl"
	"github.com/robotii/lito/vm"
)

func main() {
	versionOptionPtr := flag.Bool("v", false, "Show current Lito version")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")
	traceprofile := flag.String("trace", "", "write trace to `file`")
	inspect := flag.Bool("inspect", false, "show the generated instructions")
	machineType := flag.String("mtype", "standard", "type of the machine to use")

	flag.Parse()

	if *versionOptionPtr {
		fmt.Println(vm.Version)
		os.Exit(0)
	}

	if *traceprofile != "" {
		f, err := os.Create(*traceprofile)
		if err != nil {
			log.Fatal("could not create Trace profile: ", err)
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(f)
		_ = trace.Start(f)
		defer trace.Stop()
	}

	// CPU profiling
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(f)
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	var fp string

	switch flag.Arg(0) {
	case "":
		repl.StartREPL(vm.Version, *inspect, *machineType)
		os.Exit(0)
	default:
		fp = flag.Arg(0)

		if !strings.Contains(fp, ".") {
			flag.Usage()
			os.Exit(0)
		}
	}

	// Execute files normally
	dir, fileExt := extractFileInfo(fp)

	switch fileExt {
	case vm.FileExt:
		file := readFile(fp)
		args := flag.Args()[1:]
		instructionSets, err := compiler.CompileToInstructions(string(file), parser.NormalMode)
		reportErrorAndExit(err)

		configs := []vm.ConfigFunc{vm.Mode(parser.CommandLineMode)}
		if cfg, ok := vm.MachineConfigs[*machineType]; ok {
			configs = append(configs, cfg)
		} else {
			configs = append(configs, vm.MachineConfigs["standard"])
		}

		v, err := vm.New(dir, args, configs...)
		reportErrorAndExit(err)

		fp, err = filepath.Abs(fp)
		reportErrorAndExit(err)

		v.ExecInstructions(instructionSets, fp)
	default:
		fmt.Printf("Unknown file extension: %s\n", fileExt)
	}

	// Memory profiling
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(f)
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}

func extractFileInfo(fp string) (dir, fileExt string) {
	dir, _ = filepath.Split(fp)
	dir, _ = filepath.Abs(dir)
	fileExt = strings.TrimPrefix(filepath.Ext(fp), ".")
	return
}

func readFile(filepath string) (file []byte) {
	file, err := os.ReadFile(filepath)
	reportErrorAndExit(err)
	return
}

func reportErrorAndExit(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
