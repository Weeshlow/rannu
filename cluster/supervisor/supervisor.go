package main

import (
	"flag"
	"fmt"
	"math"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	matrix "github.com/skelterjohn/go.matrix"
	pb "github.com/unchartedsoftware/rannu/cluster/rannu"
)

var (
	host       = flag.String("addr", "127.0.0.1", "Host IP")
	basePort   = flag.Int("base-port", 10000, "Base worker port")
	numWorkers = flag.Int("workers", 2, "Number of workers")
)

func main() {
	flag.Parse()

	var err error

	conns := make([]*grpc.ClientConn, *numWorkers)
	clients := make([]pb.WorkerClient, *numWorkers)

	opts := []grpc.DialOption{grpc.WithInsecure()}

	for i := range clients {
		addr := fmt.Sprintf("%s:%d", *host, *basePort+i)
		conns[i], err = grpc.Dial(addr, opts...)
		if err != nil {
			grpclog.Fatalf("fail to dial: %v", err)
		}
		defer conns[i].Close()

		clients[i] = pb.NewWorkerClient(conns[i])
	}

	var rows, cols int
	for i := range clients {
		dataFile := &pb.DataFile{
			Name: fmt.Sprintf("%d.csv", i+1),
		}
		size, err := clients[i].LoadData(context.Background(), dataFile)
		if err != nil {
			grpclog.Fatalf("%v.LoadData() got error %v", clients[i], err)
		}
		if i == 0 {
			cols = int(size.Cols)
		} else if int(size.Cols) != cols {
			grpclog.Fatalf("Inconsistent vector sizes: %v, %v", size.Cols, cols)
		}
		rows += int(size.Rows)
	}

	sum := matrix.Zeros(1, cols)
	for i := range clients {
		subVector, err := clients[i].GetSum(context.Background(), &pb.Unit{})
		if err != nil {
			grpclog.Fatalf("%v.GetSum() got error %v", clients[i], err)
		}
		subSum := matrix.MakeDenseMatrixStacked([][]float64{subVector.Elements})
		sum.Add(subSum)
	}
	fmt.Println("sum", sum)
	sumArray := sum.Array()
	for i := range sumArray {
		sumArray[i] /= float64(rows)
	}
	fmt.Println("mean", sumArray)

	mean := &pb.Vector{
		Elements: sumArray,
	}
	scatter := matrix.Zeros(cols, cols)
	for i := range clients {
		subScatter, err := clients[i].GetScatterMatrix(context.Background(), mean)
		if err != nil {
			grpclog.Fatalf("%v.GetScatterMatrix() got error %v", clients[i], err)
		}
		vectors := make([][]float64, cols)
		for i := range subScatter.Elements {
			vectors[i] = subScatter.Elements[i].Elements
		}
		err = scatter.Add(matrix.MakeDenseMatrixStacked(vectors))
		if err != nil {
			grpclog.Fatalf("Failed to add matrices")
		}
	}
	fmt.Println("scatter", scatter)

	eigenvectors, eigenvalues, err := scatter.Eigen()
	if err != nil {
		grpclog.Fatalf("Failed to compute Eigen(): %v", err)
	}
	fmt.Println("values", eigenvalues.DiagonalCopy())
	fmt.Println("vectors", eigenvectors)
	fmt.Println("vectors T", eigenvectors.Transpose())

	topValues := []float64{-math.MaxFloat64, -math.MaxFloat64}
	topVectors := make([]*matrix.DenseMatrix, 2)
	for i, eigenvalue := range eigenvalues.DiagonalCopy() {
		if eigenvalue < topValues[1] {
			continue
		}
		if eigenvalue > topValues[0] {
			topValues[1] = topValues[0]
			topVectors[1] = topVectors[0]
			topValues[0] = eigenvalue
			topVectors[0] = eigenvectors.GetColVector(i).Transpose()
		} else {
			topValues[1] = eigenvalue
			topVectors[1] = eigenvectors.GetColVector(i).Transpose()
		}
	}
	fmt.Println("top 1", topValues[0], topVectors[0])
	fmt.Println("top 2", topValues[1], topVectors[1])

	top := &pb.Matrix{
		Elements: []*pb.Vector{
			&pb.Vector{Elements: topVectors[0].Array()},
			&pb.Vector{Elements: topVectors[1].Array()},
		},
	}
	for i := range clients {
		z, err := clients[i].ComputeScores(context.Background(), top)
		if err != nil {
			grpclog.Fatalf("%v.ComputeScores() got error %v", clients[i], err)
		}
		for j := range z.Elements {
			fmt.Println(z.Elements[j].Elements)
		}
	}
}
