/*
 * Copyright © 2022 Authors of Patu
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/redhat-et/patu/cmd/patu/daemon/kubehelper"
	"github.com/redhat-et/patu/configs"
	"github.com/redhat-et/patu/internal/bpf"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)


var rootCmd = &cobra.Command{
	Use:   "patu",
	Short: "Patu - lightweight CNI for container orchestrators managing edge devices.",
	Long: `Patu - lightweight CNI for container orchestrators managing edge devices.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if configs.Compile {
			if err = bpf.CompileEbpfProg(); err != nil {
				return fmt.Errorf(err.Error());
			}	
		}

		if err = bpf.LoadBPFMaps(); err != nil {
			return fmt.Errorf(err.Error());
		}
		
		var subnetIp net.IP
		if client := kubehelper.GetKubeClient(); client == nil {
			return fmt.Errorf("Failed to get kube client.")
		} else {
			subnetIp, _, _= kubehelper.GetSubnetFromConfig(client)
		}
		
		if subnetIp != nil {
			if err := bpf.UpdateMapWithCidrConfig(subnetIp); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("Not able to get patu subnet CIDR.")
		}

		if err = bpf.LoadAndAttachBPFProg(); err != nil {
			return fmt.Errorf(err.Error());
		}

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
		<-ch
	
		if err = bpf.UnloadBpfProg(); err != nil {
			return fmt.Errorf(err.Error());
		}
		if err = bpf.UnloadBpfMaps(); err != nil {
			return fmt.Errorf(err.Error());
		}

		return nil
	},
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       false,
		FullTimestamp:          true,
		DisableLevelTruncation: true,
		DisableColors:          true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			fs := strings.Split(f.File, "/")
			filename := fs[len(fs)-1]
			ff := strings.Split(f.Function, "/")
			_f := ff[len(ff)-1]
			return fmt.Sprintf("%s()", _f), fmt.Sprintf("%s:%d", filename, f.Line)
		},
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	log.SetReportCaller(true)

	//Flags supported by patu app.
	rootCmd.PersistentFlags().BoolVarP(&configs.Debug, "debug", "d", false, "Enable/Disable debug mode")
	rootCmd.PersistentFlags().BoolVarP(&configs.Compile, "compile", "c", false, "Enable/Disable eBPF program compilation")
}

func main() {
	execute()
}