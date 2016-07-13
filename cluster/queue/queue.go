package queue

import (
	"fmt"
	"math"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/oleiade/lane"
	matrix "github.com/skelterjohn/go.matrix"
	pb "github.com/unchartedsoftware/rannu/cluster/rannu"
)

var (
	clients    []pb.WorkerClient
	q          = lane.NewQueue()
	save       = false
	processing = false
)

type Job struct {
	Dataset         string
	Workers         int
	ResponseChannel chan *Response
}

type Response struct {
	Status       string              `json:"status"`
	Message      string              `json:"message"`
	Eigenvalues  []float64           `json:"eigenvalues"`
	Eigenvectors *matrix.DenseMatrix `json:"eigenvectors"`
	Elapsed      float64             `json:"elapsed"`
}

type sizeResponse struct {
	Size  *pb.Size
	Error error
}

type vectorResponse struct {
	Vector *pb.Vector
	Error  error
}

type matrixResponse struct {
	Matrix *pb.Matrix
	Error  error
}

type dataFileResponse struct {
	DataFile *pb.DataFile
	Error    error
}

func Listen(addrs []string, jobc chan *Job) error {
	numWorkers := len(addrs)
	conns := make([]*grpc.ClientConn, numWorkers)
	clients = make([]pb.WorkerClient, numWorkers)

	opts := []grpc.DialOption{grpc.WithInsecure()}

	var err error
	for i, addr := range addrs {
		conns[i], err = grpc.Dial(addr, opts...)
		if err != nil {
			grpclog.Printf("fail to dial: %v", err)
			return err
		}

		clients[i] = pb.NewWorkerClient(conns[i])
	}

	go func() {
		for {
			select {
			case job := <-jobc:
				grpclog.Println("Enqueuing job")
				q.Enqueue(job)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)
		for _ = range ticker.C {
			if processing {
				grpclog.Println("Still processing...")
				continue
			}
			if q.Head() != nil {
				item := q.Dequeue()
				job, ok := item.(*Job)
				if !ok {
					grpclog.Println("Invalid job!")
					continue
				}
				processing = true
				process(job)
				continue
			}
		}
	}()

	return nil
}

func process(job *Job) {
	var err error

	resp := &Response{}

	if job.Workers > len(clients) {
		grpclog.Printf("Invalid worker number: %v > %v", job.Workers, len(clients))
		resp.Message = "Invalid worker number"
		resp.Status = "error"
		job.ResponseChannel <- resp
		processing = false
		return
	}

	grpclog.Println("Processing job")
	startTime := time.Now()

	sizec := make(chan sizeResponse)
	var rows, cols int
	for i := 0; i < job.Workers; i++ {
		dataFile := &pb.DataFile{
			Name: fmt.Sprintf("%s-%d-%d.csv", job.Dataset, job.Workers, i+1),
		}
		go func(client pb.WorkerClient) {
			size, err := client.LoadData(context.Background(), dataFile)
			sizec <- sizeResponse{
				Size:  size,
				Error: err,
			}
		}(clients[i])
	}
	for i := 0; i < job.Workers; i++ {
		sizeResp := <-sizec
		size := sizeResp.Size
		err := sizeResp.Error
		if err != nil {
			grpclog.Printf("%v.LoadData() got error %v", clients[i], err)
			resp.Message = "Could not load data"
			resp.Status = "error"
			job.ResponseChannel <- resp
			processing = false
			return
		}
		if i == 0 {
			cols = int(size.Cols)
		} else if int(size.Cols) != cols {
			grpclog.Printf("Inconsistent vector sizes: %v, %v", size.Cols, cols)
			resp.Message = "Inconsistent vectors sizes"
			resp.Status = "error"
			job.ResponseChannel <- resp
			processing = false
			return
		}
		rows += int(size.Rows)
	}

	sumc := make(chan vectorResponse)
	sum := matrix.Zeros(1, cols)
	for i := 0; i < job.Workers; i++ {
		go func(client pb.WorkerClient) {
			vector, err := client.GetSum(context.Background(), &pb.Unit{})
			sumc <- vectorResponse{
				Vector: vector,
				Error:  err,
			}
		}(clients[i])
	}
	for i := 0; i < job.Workers; i++ {
		vectorResp := <-sumc
		subVector := vectorResp.Vector
		err := vectorResp.Error
		if err != nil {
			grpclog.Printf("%v.GetSum() got error %v", clients[i], err)
			resp.Message = "Could not get sum"
			resp.Status = "error"
			job.ResponseChannel <- resp
			processing = false
			return
		}
		subSum := matrix.MakeDenseMatrixStacked([][]float64{subVector.Elements})
		sum.Add(subSum)
	}
	sumArray := sum.Array()
	for i := range sumArray {
		sumArray[i] /= float64(rows)
	}

	matrixc := make(chan matrixResponse)
	mean := &pb.Vector{
		Elements: sumArray,
	}
	scatter := matrix.Zeros(cols, cols)
	for i := 0; i < job.Workers; i++ {
		go func(client pb.WorkerClient) {
			matrix, err := client.GetScatterMatrix(context.Background(), mean)
			matrixc <- matrixResponse{
				Matrix: matrix,
				Error:  err,
			}
		}(clients[i])
	}
	for i := 0; i < job.Workers; i++ {
		matrixResp := <-matrixc
		subScatter := matrixResp.Matrix
		err := matrixResp.Error
		if err != nil {
			grpclog.Printf("%v.GetScatterMatrix() got error %v", clients[i], err)
			resp.Message = "Could not get scatter matrix"
			resp.Status = "error"
			job.ResponseChannel <- resp
			processing = false
			return
		}
		vectors := make([][]float64, cols)
		for i := range subScatter.Elements {
			vectors[i] = subScatter.Elements[i].Elements
		}
		err = scatter.Add(matrix.MakeDenseMatrixStacked(vectors))
		if err != nil {
			grpclog.Printf("Failed to add matrices")
			resp.Message = "Failed to add matrices"
			resp.Status = "error"
			job.ResponseChannel <- resp
			processing = false
			return
		}
	}

	eigenvectors, eigenvaluesMatrix, err := scatter.Eigen()
	if err != nil {
		grpclog.Printf("Failed to compute Eigen(): %v", err)
		resp.Message = "Could not compute eigenvalues/vectors"
		resp.Status = "error"
		job.ResponseChannel <- resp
		processing = false
		return
	}
	resp.Eigenvalues = eigenvaluesMatrix.DiagonalCopy()
	resp.Eigenvectors = eigenvectors

	topValues := []float64{-math.MaxFloat64, -math.MaxFloat64}
	topVectors := make([]*matrix.DenseMatrix, 2)
	for i, eigenvalue := range resp.Eigenvalues {
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

	if save {
		top := &pb.Matrix{
			Elements: []*pb.Vector{
				&pb.Vector{Elements: topVectors[0].Array()},
				&pb.Vector{Elements: topVectors[1].Array()},
			},
		}
		filec := make(chan dataFileResponse)
		for i := 0; i < job.Workers; i++ {
			go func(client pb.WorkerClient) {
				dataFile, err := client.ComputeScores(context.Background(), top)
				filec <- dataFileResponse{
					DataFile: dataFile,
					Error:    err,
				}
			}(clients[i])
		}
		for i := 0; i < job.Workers; i++ {
			fileResp := <-filec
			err := fileResp.Error
			if err != nil {
				grpclog.Printf("%v.ComputeScores() got error %v", clients[i], err)
				resp.Message = "Could not compute scores"
				resp.Status = "error"
				job.ResponseChannel <- resp
				processing = false
				return
			}
		}
	}

	endTime := time.Now()
	resp.Elapsed = endTime.Sub(startTime).Seconds()
	resp.Status = "ok"
	job.ResponseChannel <- resp

	processing = false
}
