package main

import (
	"bufio"
	"fmt"
	"github.com/docker/go-connections/nat"
	"io"
	"log"
	"os"

	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"strings"
	"time"
)

type StatsDecode struct {
	// Common stats
	Read    time.Time `json:"read"`
	PreRead time.Time `json:"preread"`

	// Linux specific stats, not populated on Windows.
	PidsStats  types.PidsStats  `json:"pids_stats,omitempty"`
	BlkioStats types.BlkioStats `json:"blkio_stats,omitempty"`

	// Windows specific stats, not populated on Linux.
	NumProcs     uint32             `json:"num_procs"`
	StorageStats types.StorageStats `json:"storage_stats,omitempty"`

	// Shared stats
	CPUStats    types.CPUStats    `json:"cpu_stats,omitempty"`
	PreCPUStats types.CPUStats    `json:"precpu_stats,omitempty"`
	MemoryStats types.MemoryStats `json:"memory_stats,omitempty"`

	Networks map[string]types.NetworkStats `json:"networks,omitempty"`
}

type StatsOut struct {
	MemPercent       float64
	PreviousCPU      uint64
	PreviousSystem   uint64
	CpuPercent       float64
	BlkRead          uint64
	BlkWrite         uint64
	Mem              uint64
	MemLimit         uint64
	PidsStatsCurrent uint64
	NetRx            float64
	NetTx            float64
}

type DockerStats struct {
	ToDecode    types.ContainerStats
	statsDecode StatsDecode
	Stats       StatsOut
}

func (el *DockerStats) Decode() {
	decoder := json.NewDecoder(el.ToDecode.Body)
	decoder.Decode(&el.statsDecode)

	el.Stats.MemPercent = 0.0
	if el.statsDecode.MemoryStats.Limit != 0 {
		el.Stats.MemPercent = float64(el.statsDecode.MemoryStats.Usage) /
			float64(el.statsDecode.MemoryStats.Limit) * 100.0
	}
	el.Stats.PreviousCPU = el.statsDecode.PreCPUStats.CPUUsage.TotalUsage
	el.Stats.PreviousSystem = el.statsDecode.PreCPUStats.SystemUsage
	el.Stats.CpuPercent = el.calculateCPUPercentUnix()
	el.Stats.BlkRead, el.Stats.BlkWrite = el.calculateBlockIO()
	el.Stats.Mem = el.statsDecode.MemoryStats.Usage
	el.Stats.MemLimit = el.statsDecode.MemoryStats.Limit
	el.Stats.PidsStatsCurrent = el.statsDecode.PidsStats.Current
	el.Stats.NetRx, el.Stats.NetTx = el.calculateNetwork()
}

func (el *DockerStats) calculateCPUPercentUnix() float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in
		// between readings
		cpuDelta = float64(el.statsDecode.CPUStats.CPUUsage.TotalUsage) -
			float64(el.Stats.PreviousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(el.statsDecode.CPUStats.SystemUsage) -
			float64(el.Stats.PreviousSystem)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) *
			float64(len(el.statsDecode.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}

func (el *DockerStats) calculateBlockIO() (blkRead uint64, blkWrite uint64) {
	for _, bioEntry := range el.statsDecode.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(bioEntry.Op) {
		case "read":
			blkRead = blkRead + bioEntry.Value
		case "write":
			blkWrite = blkWrite + bioEntry.Value
		}
	}
	return
}

func (el *DockerStats) calculateNetwork() (float64, float64) {
	var rx, tx float64

	for _, v := range el.statsDecode.Networks {
		rx += float64(v.RxBytes)
		tx += float64(v.TxBytes)
	}
	return rx, tx
}

func main() {

	var sumary []types.ImageSummary
	var reader io.ReadCloser
	var insp types.ImageInspect
	var data []byte
	var list []types.Container

	img := "ghost:latest"

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion("1.39"))
	if err != nil {
		panic(err)
	}

	list, err = cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	bJson, _ := json.Marshal(&list)
	fmt.Printf("%s\n", bJson)
	fmt.Println()

	timeOut := 10 * time.Second
	for _, containerData := range list {
		if strings.Contains(img, containerData.Image) && containerData.State == "running" {
			err = cli.ContainerStop(ctx, containerData.ID, &timeOut)
			if err != nil {
				log.Panic(err.Error())
			}
		}

		if strings.Contains(img, containerData.Image) && containerData.State != "running" {
			err = cli.ContainerRemove(ctx, containerData.ID, types.ContainerRemoveOptions{Force: true})
			if err != nil {
				log.Panic(err.Error())
			}
		}
	}

	reader, err = cli.ImagePull(ctx, img, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fmt.Printf("%s\n", scanner.Bytes())
	}
	if err = scanner.Err(); err != nil {
		log.Fatal(err.Error())
	}

	sumary, err = cli.ImageList(ctx, types.ImageListOptions{All: true})
	if err != nil {
		log.Panic(err.Error())
	}

	for _, imageSumary := range sumary {
		for _, tagContent := range imageSumary.RepoTags {
			if strings.Contains(tagContent, img) {

				insp, data, err = cli.ImageInspectWithRaw(ctx, imageSumary.ID)
				if err != nil {
					log.Panic(err.Error())
				}

				bJson, _ = json.Marshal(&insp)
				fmt.Printf("%s\n", bJson)
				fmt.Printf("%s\n", data)
				fmt.Println()

			}
		}
	}

	config := &container.Config{
		Image: img,
		ExposedPorts: nat.PortSet{
			"8080/tcp": struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"2368/tcp": []nat.PortBinding{
				{
					HostPort: "8080",
				},
			},
		},
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID,
		types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	var stats DockerStats

	stats.ToDecode, err = cli.ContainerStats(ctx, resp.ID, false)
	if err != nil {
		panic(err)
	}

	go func(s DockerStats) {
		for {
			fmt.Printf("PreviousCPU: %v\n", s.Stats.PreviousCPU)
			fmt.Printf("CpuPercent: %v\n", s.Stats.CpuPercent)
			fmt.Printf("BlkRead: %v\n", s.Stats.BlkRead)
			fmt.Printf("BlkWrite: %v\n", s.Stats.BlkWrite)
			fmt.Printf("Mem: %v\n", s.Stats.Mem)
			fmt.Printf("MemLimit: %v\n", s.Stats.MemLimit)
			fmt.Printf("MemPercent: %v\n", s.Stats.MemPercent)
			fmt.Printf("NetRx: %v\n", s.Stats.NetRx)
			fmt.Printf("NetTx: %v\n", s.Stats.NetTx)
			fmt.Printf("PidsStatsCurrent: %v\n", s.Stats.PidsStatsCurrent)
			fmt.Printf("PreviousSystem: %v\n\n", s.Stats.PreviousSystem)
			time.Sleep(100 * time.Millisecond)
		}
	}(stats)

	out, err := cli.ContainerLogs(ctx, resp.ID,
		types.ContainerLogsOptions{ShowStdout: true, Details: true, Follow: true, ShowStderr: true, Timestamps: true})
	if err != nil {
		panic(err)
	}

	io.Copy(os.Stdout, out)

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	list, err = cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	bJson, _ = json.Marshal(&list)
	fmt.Printf("%s\n", bJson)
	fmt.Println()
}
