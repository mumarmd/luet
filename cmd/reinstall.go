// Copyright © 2021 Ettore Di Giacinto <mudler@mocaccino.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.
package cmd

import (
	installer "github.com/mudler/luet/pkg/installer"
	"github.com/mudler/luet/pkg/solver"

	helpers "github.com/mudler/luet/cmd/helpers"
	. "github.com/mudler/luet/pkg/config"
	. "github.com/mudler/luet/pkg/logger"
	pkg "github.com/mudler/luet/pkg/package"

	"github.com/spf13/cobra"
)

var reinstallCmd = &cobra.Command{
	Use:   "reinstall <pkg1> <pkg2> <pkg3>",
	Short: "reinstall a set of packages",
	Long: `Reinstall a group of packages in the system:

	$ luet reinstall -y system/busybox shells/bash system/coreutils ...
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		LuetCfg.Viper.BindPFlag("system.database_path", cmd.Flags().Lookup("system-dbpath"))
		LuetCfg.Viper.BindPFlag("system.database_engine", cmd.Flags().Lookup("system-engine"))
		LuetCfg.Viper.BindPFlag("system.rootfs", cmd.Flags().Lookup("system-target"))
		LuetCfg.Viper.BindPFlag("solver.type", cmd.Flags().Lookup("solver-type"))
		LuetCfg.Viper.BindPFlag("solver.discount", cmd.Flags().Lookup("solver-discount"))
		LuetCfg.Viper.BindPFlag("solver.rate", cmd.Flags().Lookup("solver-rate"))
		LuetCfg.Viper.BindPFlag("solver.max_attempts", cmd.Flags().Lookup("solver-attempts"))
		LuetCfg.Viper.BindPFlag("onlydeps", cmd.Flags().Lookup("onlydeps"))
		LuetCfg.Viper.BindPFlag("force", cmd.Flags().Lookup("force"))
		LuetCfg.Viper.BindPFlag("for", cmd.Flags().Lookup("for"))

		LuetCfg.Viper.BindPFlag("yes", cmd.Flags().Lookup("yes"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		var toUninstall pkg.Packages
		var toAdd pkg.Packages

		stype := LuetCfg.Viper.GetString("solver.type")
		discount := LuetCfg.Viper.GetFloat64("solver.discount")
		rate := LuetCfg.Viper.GetFloat64("solver.rate")
		attempts := LuetCfg.Viper.GetInt("solver.max_attempts")
		force := LuetCfg.Viper.GetBool("force")
		onlydeps := LuetCfg.Viper.GetBool("onlydeps")
		concurrent, _ := cmd.Flags().GetBool("solver-concurrent")
		yes := LuetCfg.Viper.GetBool("yes")
		dbpath := LuetCfg.Viper.GetString("system.database_path")
		rootfs := LuetCfg.Viper.GetString("system.rootfs")
		engine := LuetCfg.Viper.GetString("system.database_engine")
		downloadOnly, _ := cmd.Flags().GetBool("download-only")

		LuetCfg.System.DatabaseEngine = engine
		LuetCfg.System.DatabasePath = dbpath
		LuetCfg.System.Rootfs = rootfs

		for _, a := range args {
			pack, err := helpers.ParsePackageStr(a)
			if err != nil {
				Fatal("Invalid package string ", a, ": ", err.Error())
			}
			toUninstall = append(toUninstall, pack)
			toAdd = append(toAdd, pack)
		}

		// This shouldn't be necessary, but we need to unmarshal the repositories to a concrete struct, thus we need to port them back to the Repositories type
		repos := installer.Repositories{}
		for _, repo := range LuetCfg.SystemRepositories {
			if !repo.Enable {
				continue
			}
			r := installer.NewSystemRepository(repo)
			repos = append(repos, r)
		}

		LuetCfg.GetSolverOptions().Type = stype
		LuetCfg.GetSolverOptions().LearnRate = float32(rate)
		LuetCfg.GetSolverOptions().Discount = float32(discount)
		LuetCfg.GetSolverOptions().MaxAttempts = attempts

		if concurrent {
			LuetCfg.GetSolverOptions().Implementation = solver.ParallelSimple
		} else {
			LuetCfg.GetSolverOptions().Implementation = solver.SingleCoreSimple
		}

		Debug("Solver", LuetCfg.GetSolverOptions().CompactString())

		// Load config protect configs
		installer.LoadConfigProtectConfs(LuetCfg)

		inst := installer.NewLuetInstaller(installer.LuetInstallerOptions{
			Concurrency:                 LuetCfg.GetGeneral().Concurrency,
			SolverOptions:               *LuetCfg.GetSolverOptions(),
			NoDeps:                      true,
			Force:                       force,
			OnlyDeps:                    onlydeps,
			PreserveSystemEssentialData: true,
			Ask:                         !yes,
			DownloadOnly:                downloadOnly,
		})
		inst.Repositories(repos)

		system := &installer.System{Database: LuetCfg.GetSystemDB(), Target: LuetCfg.GetSystem().Rootfs}
		err := inst.Swap(toUninstall, toAdd, system)
		if err != nil {
			Fatal("Error: " + err.Error())
		}
	},
}

func init() {

	reinstallCmd.Flags().String("system-dbpath", "", "System db path")
	reinstallCmd.Flags().String("system-target", "", "System rootpath")
	reinstallCmd.Flags().String("system-engine", "", "System DB engine")

	reinstallCmd.Flags().String("solver-type", "", "Solver strategy ( Defaults none, available: "+AvailableResolvers+" )")
	reinstallCmd.Flags().Float32("solver-rate", 0.7, "Solver learning rate")
	reinstallCmd.Flags().Float32("solver-discount", 1.0, "Solver discount rate")
	reinstallCmd.Flags().Int("solver-attempts", 9000, "Solver maximum attempts")
	reinstallCmd.Flags().Bool("onlydeps", false, "Consider **only** package dependencies")
	reinstallCmd.Flags().Bool("force", false, "Skip errors and keep going (potentially harmful)")
	reinstallCmd.Flags().Bool("solver-concurrent", false, "Use concurrent solver (experimental)")
	reinstallCmd.Flags().BoolP("yes", "y", false, "Don't ask questions")
	reinstallCmd.Flags().Bool("download-only", false, "Download only")

	RootCmd.AddCommand(reinstallCmd)
}