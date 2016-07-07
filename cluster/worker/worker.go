package main

import (
	"errors"
	"fmt"
	"io"

	"golang.org/x/net/context"

	"github.com/montanaflynn/stats"
	matrix "github.com/skelterjohn/go.matrix"
	pb "github.com/unchartedsoftware/rannu/cluster/rannu"
)

type workerServer struct {
	matrix *matrix.DenseMatrix
}

func (w *workerServer) RecordVectors(stream pb.Worker_RecordVectorsServer) error {
	size := 0
	vectors := [][]float64{}

	for {
		vector, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.Unit{})
		}
		if err != nil {
			return err
		}
		if size == 0 {
			size = len(vector.Elements)
		} else if len(vector.Elements) != size {
			return errors.New("Inconsistent vector sizes")
		}
		vectors = append(vectors, vector.Elements)
	}

	w.matrix = matrix.MakeDenseMatrixStacked(vectors)

	return nil
}

func (w *workerServer) GetMean(ctx context.Context, _ *pb.Unit) (*pb.Vector, error) {
	mean := &pb.Vector{
		Elements: make([]float64, w.matrix.Cols()),
	}

	var err error
	for i := range mean.Elements {
		col := w.matrix.GetColVector(i).Transpose()
		mean.Elements[i], err = stats.Mean(col.Array())
		if err != nil {
			return nil, err
		}
	}

	return mean, nil
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

func main() {
	w := &workerServer{}
	w.matrix = matrix.MakeDenseMatrixStacked([][]float64{
		[]float64{1, 2, 3},
		[]float64{3, 6, 9},
		[]float64{2, 4, 6},
	})
	fmt.Println(w.matrix)
	mean, err := w.GetMean(nil, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(mean)
	fmt.Println(w.GetScatterMatrix(nil, mean))
}
