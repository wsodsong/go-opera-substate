// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

//  counts storage sizes of accounts in substates
//  and changes between the in-storage and out-storage
//  of an account. 

package main

import (
	"flag"
	"fmt"
	"time"
	"strings"
	
	"os"
	"encoding/csv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
)

type CodeStats struct {
	Max int
	Min int
	Avg float64
	Count uint64
}

var (
	firstBlock = flag.Uint64("first-block",1, "first block of blockchain")
	lastBlock = flag.Uint64("last-block",9000000, "last block of blockchain")
	verboseOpt = flag.Bool("verbose", false, "verbose")
)

func main() {
        fmt.Println("metric: start.")

	codeSizes := make(map[common.Address]CodeStats)
	maxNonce := make(map[common.Address]uint64)
	flag.Parse()
        first := uint64(*firstBlock)
        last := uint64(*lastBlock)
	verbose := bool(*verboseOpt)

	fmt.Printf("metric: from block %v to block %v\n", first, last)

	research.OpenSubstateDBReadOnly()
	defer research.CloseSubstateDB()
	//contractCreationMap = make(map[uint64]uint64)

	start := time.Now()
	for block := uint64(first); block <= uint64(last); block++ {
		if block%100000 == 0 {
			duration := time.Since(start) + 1*time.Nanosecond
			fmt.Printf("metric: elapsed time: %v, number = %v\n", duration.Round(1*time.Millisecond), block)
		}
		for tx := 0; ; tx++ {
			if !research.HasSubstate(block, tx) {
				break
			}
			substate := research.GetSubstate(block, tx)
			for wallet, inputAccount := range substate.InputAlloc {
				if stat, found := codeSizes[wallet];  !found {
					codeSizes[wallet] = CodeStats{len(inputAccount.Code), len(inputAccount.Code), float64(len(inputAccount.Code)), 1}
				} else {
					newCodeSize := len(inputAccount.Code)
					stat.Avg = (stat.Avg * float64(stat.Count) + float64(newCodeSize)) / (float64(stat.Count)  + 1)
					stat.Count++
					if stat.Max < len(inputAccount.Code) {
						stat.Max = len(inputAccount.Code)
					} else if stat.Min > len(inputAccount.Code) {
						stat.Min = len(inputAccount.Code)
					}
					codeSizes[wallet] = stat
				}
				if val, found := maxNonce[wallet];  !found || val < inputAccount.Nonce {
					maxNonce[wallet] = inputAccount.Nonce
				}
				if verbose {
					fmt.Printf("metric: data %v %v %v\tcode %d,%d %d\tnonce %d->%d\n", 
							block, 
							tx,
							wallet.Hex(), 
							codeSizes[wallet].Max,
							codeSizes[wallet].Min,
							codeSizes[wallet].Count,
							inputAccount.Nonce,
							maxNonce[wallet])
					}
			}
		}
	}
	codeSizeFile, err := os.Create("codesize.csv")
	if err != nil {
		fmt.Errorf("failed to create file %s", err)
	}
	codeSizeWriter := csv.NewWriter(codeSizeFile)
	for wallet, codesize := range codeSizes {
		err := codeSizeWriter.Write([]string{fmt.Sprintf("%v",
						strings.ToLower(wallet.Hex())), 
						fmt.Sprintf("%d", codesize.Max),
						fmt.Sprintf("%d", codesize.Min),
						fmt.Sprintf("%.0f", codesize.Avg),
						fmt.Sprintf("%d", codesize.Count),
					})
		//fmt.Printf("%v,%d\n", strings.ToLower(wallet.Hex()), codesize)
		if err != nil {
			panic(err)
		}
	}
	codeSizeWriter.Flush()
	nonceFile, err := os.Create("nonce.csv")
	if err != nil {
		fmt.Errorf("failed to create file %s", err)
	}
	nonceWriter := csv.NewWriter(nonceFile)
	for wallet, nonce := range maxNonce {
		err := nonceWriter.Write([]string{fmt.Sprintf("%v",strings.ToLower(wallet.Hex())), fmt.Sprintf("%d", nonce)})
		//fmt.Printf("%v,%d\n", strings.ToLower(wallet.Hex()), nonce)
		if err != nil {
			panic(err)
		}
	}
	nonceWriter.Flush()
        //research.CloseSubstateDB()
        fmt.Println("metric: end.")
}

