package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	_ "os"
	"strconv"
	"strings"
	"time"

	"github.com/BrentG-1849260/pack3d/pack3d"
	"github.com/fogleman/fauxgl"
)

const (
	bvhDetail           = 8
	annealingIterations = 2000000
)

func timed(name string) func() {
	if len(name) > 0 {
		fmt.Printf("%s... ", name)
	}
	start := time.Now()
	return func() {
		fmt.Println(time.Since(start))
	}
}

func main() {
	outputPathPtr := flag.String("output_path", "packing.stl", "Path to the output stl file.")
	execTimePtr := flag.Int("exec_time", 180, "Stop after approximately this amount of seconds.")
	rotAllowedPtr := flag.String("rot", "", "Comma-separated list of booleans to disable/enable rotations for each object. E.g. for 3 objects: -rot=1,0,1 where 0 means rotation is not allowed. (all enabled by default)")
	flag.Parse()

	flagsOk := true
	rotationAllowance := make([]bool, len(flag.Args()))
	for i, _ := range rotationAllowance {
		rotationAllowance[i] = true
	}
	if *rotAllowedPtr != "" {
		parts := strings.Split(*rotAllowedPtr, ",")
		for i, rotAllowed := range parts {
			b, err := strconv.ParseBool(rotAllowed)
			if err != nil {
				flagsOk = false
				break
			}
			if i >= len(rotationAllowance) {
				break
			}
			rotationAllowance[i] = b
		}
	}

	var done func()

	rand.Seed(time.Now().UTC().UnixNano())

	model := pack3d.NewModel()
	count := 1
	ok := false
	var totalVolume float64
	for i, arg := range flag.Args() {
		_count, err := strconv.ParseInt(arg, 0, 0)
		if err == nil {
			count = int(_count)
			continue
		}

		done = timed(fmt.Sprintf("loading mesh %s", arg))
		mesh, err := fauxgl.LoadMesh(arg)
		if err != nil {
			panic(err)
		}
		done()

		totalVolume += mesh.BoundingBox().Volume()
		size := mesh.BoundingBox().Size()
		fmt.Printf("  %d triangles\n", len(mesh.Triangles))
		fmt.Printf("  %g x %g x %g\n", size.X, size.Y, size.Z)

		done = timed("centering mesh")
		mesh.Center()
		done()

		done = timed("building bvh tree")
		fmt.Println(i)
		fmt.Println(rotationAllowance)
		model.Add(mesh, bvhDetail, count, rotationAllowance[i-1])
		ok = true
		done()
	}

	if !ok || !flagsOk {
		fmt.Println("Usage: pack3d -output_path=path/to/output.stl -exec_time=180 -rot=1,0... N1 mesh1.stl N2 mesh2.stl ...")
		fmt.Println(" - Packs N copies of each mesh into as small of a volume as possible.")
		fmt.Println(" - Runs for approximately exec_time seconds.")
		fmt.Println(" - Rotations for each object are disabled/enabled using -rot.")
		fmt.Println(" - Results are written to disk (at output_path) whenever a new best is found.")
		return
	}

	side := math.Pow(totalVolume, 1.0/3)
	model.Deviation = side / 32

	best := 1e9
	startTime := time.Now()
	for {
		model = model.Pack(annealingIterations, nil)
		score := model.Energy()
		if score < best {
			best = score
			done = timed("writing mesh")
			model.Mesh().SaveSTL(fmt.Sprintf(*outputPathPtr))
			// model.TreeMesh().SaveSTL(fmt.Sprintf("out%dtree.stl", int(score*100000)))
			done()
		}
		model.Reset()
		if int(time.Now().Sub(startTime).Seconds()) > *execTimePtr {
			break
		}
	}
}
