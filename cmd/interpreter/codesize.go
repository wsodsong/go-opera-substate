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
	Count uint64
	Nonce uint64
	InCreateTx bool
	InCallTx bool
	InTransferTx bool
}

var (
	firstBlock = flag.Uint64("first-block",1, "first block of blockchain")
	lastBlock = flag.Uint64("last-block",9000000, "last block of blockchain")
	verboseOpt = flag.Bool("verbose", false, "verbose")
)

const (
	CreateTx   int = iota
	TransferTx int = iota
	CallTx 	   int = iota
	UnknownTx  int = iota
)

func GetTxType (to *common.Address, alloc substate.SubstateAlloc) int{
	if (to == nil) {
		return CreateTx
	} 
	account, hasReceiver := alloc[*to]
	if (to != nil && (!hasReceiver || len(account.Code) == 0)) {
		return TransferTx
	}
	if (to != nil && (hasReceiver && len(account.Code) > 0)) {
		return CallTx
	}
	return  UnknownTx
}

func Btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func main() {
        fmt.Println("metric: start.")

	accountStats := make(map[common.Address]CodeStats)
	flag.Parse()
        first := uint64(*firstBlock)
        last := uint64(*lastBlock)
	verbose := bool(*verboseOpt)

	fmt.Printf("metric: from block %v to block %v\n", first, last)

	substate.OpenSubstateDBReadOnly()
	defer substate.CloseSubstateDB()
	//contractCreationMap = make(map[uint64]uint64)

	start := time.Now()
	for block := uint64(first); block <= uint64(last); block++ {
		if block%100000 == 0 {
			duration := time.Since(start) + 1*time.Nanosecond
			fmt.Printf("metric: elapsed time: %v, number = %v\n", duration.Round(1*time.Millisecond), block)
		}
		for tx := 0; ; tx++ {
			if !substate.HasSubstate(block, tx) {
				break
			}
			ss := substate.GetSubstate(block, tx)
			to := ss.Message.To
			txType := GetTxType (to, ss.InputAlloc)
			for wallet, inputAccount := range ss.InputAlloc {
				newCodeSize := len(inputAccount.Code)
				if stat, found := accountStats[wallet];  !found {
					accountStats[wallet] = CodeStats{newCodeSize, 
									newCodeSize, 
									1, 
									inputAccount.Nonce,
									txType == CreateTx,
									txType == TransferTx,
									txType == CallTx}
				} else {
					stat.Count++
					stat.InCreateTx = stat.InCreateTx || (txType == CreateTx)
					stat.InTransferTx = stat.InTransferTx || (txType == TransferTx)
					stat.InCallTx   = stat.InCallTx || (txType == CallTx)
					if stat.Max < newCodeSize {
						stat.Max = newCodeSize
					} else if stat.Min > newCodeSize {
						stat.Min = newCodeSize
					}
					if stat.Nonce < inputAccount.Nonce {
						stat.Nonce = inputAccount.Nonce
					}
					accountStats[wallet] = stat
				}

				if verbose {
					fmt.Printf("%v,%v,%v,%d,%d,%d,%d,%d\n",
							block,
							tx,
							wallet.Hex(),
							accountStats[wallet].Max,
							accountStats[wallet].Min,
							accountStats[wallet].Count,
							accountStats[wallet].Nonce,
							txType,
						)
				}
			}
		}
	}
	codeSizeFile, err := os.Create("codesize.csv")
	if err != nil {
		fmt.Errorf("failed to create file %s", err)
	}
	codeSizeWriter := csv.NewWriter(codeSizeFile)
	for wallet, codesize := range accountStats {
		
		err := codeSizeWriter.Write([]string{fmt.Sprintf("%v",
						strings.ToLower(wallet.Hex())), 
						fmt.Sprintf("%d", codesize.Max),
						fmt.Sprintf("%d", codesize.Min),
						fmt.Sprintf("%d", codesize.Count),
						fmt.Sprintf("%d", codesize.Nonce),
						fmt.Sprintf("%d", Btoi(codesize.InCreateTx)),
						fmt.Sprintf("%d", Btoi(codesize.InTransferTx)),
						fmt.Sprintf("%d", Btoi(codesize.InCallTx)),
					})
		if err != nil {
			panic(err)
		}
	}
	codeSizeWriter.Flush()
        fmt.Println("metric: end.")
}

