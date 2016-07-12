package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"golang.org/x/net/context"

	"github.com/montanaflynn/stats"
	matrix "github.com/skelterjohn/go.matrix"
	pb "github.com/unchartedsoftware/rannu/cluster/rannu"
)

var (
	port = flag.Int("port", 10000, "The server port")
)

type workerServer struct {
	filename string
	matrix   *matrix.DenseMatrix
}

func (w *workerServer) LoadData(ctx context.Context, file *pb.DataFile) (*pb.Size, error) {
	grpclog.Printf("Processing %s...", file.Name)
	w.filename = file.Name

	cols := 0
	vectors := [][]float64{}

	f, err := os.Open(fmt.Sprintf("data/%s", file.Name))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(bufio.NewReader(f))
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}

		num := len(row)
		if cols == 0 {
			cols = num
		} else if num != cols {
			return nil, errors.New("Inconsistent vector sizes")
		}

		vector := make([]float64, num)
		for i := range vector {
			vector[i], err = strconv.ParseFloat(row[i], 64)
			if err != nil {
				return nil, err
			}
		}

		vectors = append(vectors, vector)
	}

	w.matrix = matrix.MakeDenseMatrixStacked(vectors)
	grpclog.Printf("Processed %d x %d matrix", len(vectors), cols)

	size := &pb.Size{
		Rows: int32(len(vectors)),
		Cols: int32(cols),
	}
	return size, nil
}

func (w *workerServer) GetSum(ctx context.Context, unit *pb.Unit) (*pb.Vector, error) {
	if w.matrix == nil {
		return nil, errors.New("No matrix available")
	}
	sum := &pb.Vector{
		Elements: make([]float64, w.matrix.Cols()),
	}

	var err error
	for i := range sum.Elements {
		col := w.matrix.GetColVector(i).Transpose()
		sum.Elements[i], err = stats.Sum(col.Array())
		if err != nil {
			return nil, err
		}
	}

	return sum, nil
}

func (w *workerServer) GetScatterMatrix(ctx context.Context, mean *pb.Vector) (*pb.Matrix, error) {
	numRows, numCols := w.matrix.GetSize()

	rows := make([][]float64, numRows)
	for i := range rows {
		rows[i] = mean.Elements
	}

	meanMatrix := matrix.MakeDenseMatrixStacked(rows)

	err := w.matrix.SubtractDense(meanMatrix)
	if err != nil {
		return nil, err
	}

	scatter := matrix.Zeros(numCols, numCols)

	for i := 0; i < numRows; i++ {
		row := w.matrix.GetRowVector(i)
		rowT := row.Transpose()
		prod, err := rowT.TimesDense(row)
		if err != nil {
			return nil, err
		}
		err = scatter.AddDense(prod)
		if err != nil {
			return nil, err
		}
	}

	vectors := make([]*pb.Vector, numCols)
	for i := 0; i < numCols; i++ {
		vectors[i] = &pb.Vector{
			Elements: scatter.GetRowVector(i).Array(),
		}
	}

	mat := &pb.Matrix{
		Elements: vectors,
	}

	return mat, nil
}

func (w *workerServer) ComputeScores(ctx context.Context, top *pb.Matrix) (*pb.DataFile, error) {
	k := len(top.Elements)
	topVectors := make([][]float64, k)
	for i := range topVectors {
		topVectors[i] = top.Elements[i].Elements
	}
	p := matrix.MakeDenseMatrixStacked(topVectors)

	vectors := make([][]float64, w.matrix.Rows())
	for i := range vectors {
		vector, err := p.TimesDense(w.matrix.GetRowVector(i).Transpose())
		if err != nil {
			return nil, err
		}
		vectors[i] = vector.Transpose().Array()
	}

	in, err := os.Open(fmt.Sprintf("data/answers-%s", w.filename))
	if err != nil {
		return nil, err
	}
	defer in.Close()

	answers := []float64{}
	r := csv.NewReader(bufio.NewReader(in))
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}

		if len(row) != 1 {
			return nil, errors.New("Inconsistent answer vector size")
		}

		answer, err := strconv.ParseFloat(row[0], 64)
		if err != nil {
			return nil, err
		}

		answers = append(answers, answer)
	}

	if len(answers) != len(vectors) {
		return nil, errors.New("Inconsistent answer and vector sizes")
	}

	filename := fmt.Sprintf("data/projected-%s", w.filename)
	out, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	var values []string
	wr := csv.NewWriter(out)
	for i := range answers {
		vectors[i] = append(vectors[i], answers[i])

		values = []string{}
		for _, value := range vectors[i] {
			values = append(values, strconv.FormatFloat(value, 'E', -1, 32))
		}
		if err = wr.Write(values); err != nil {
			return nil, err
		}
	}
	wr.Flush()
	if err = wr.Error(); err != nil {
		return nil, err
	}

	return &pb.DataFile{Name: filename}, nil
}

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		grpclog.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterWorkerServer(grpcServer, new(workerServer))

	grpclog.Printf("Listening on %d", *port)
	grpcServer.Serve(lis)
}
