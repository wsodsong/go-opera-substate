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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/substate"
)

var (
	firstBlock = flag.Uint64("first-block",1, "first block of blockchain")
	lastBlock = flag.Uint64("last-block",9000000, "last block of blockchain")
)

// computeStorageSize computes the number of non-zero storage entries
func computeStorageSizes(inUpdateSet map[common.Hash]common.Hash, outUpdateSet map[common.Hash]common.Hash) (int64, uint64, uint64) {
	deltaSize := int64(0)
	inUpdateSize := uint64(0)
	outUpdateSize := uint64(0)
	for address, outValue := range outUpdateSet {
		if inValue, found := inUpdateSet[address]; found {
			if (inValue == common.Hash{} && outValue != common.Hash{}) {
				// storage increases by one new cell
				// (cell is empty in in-storage)
				deltaSize ++
			} else if(inValue != common.Hash{} && outValue == common.Hash{}) {
				// storage shrinks by one new cell
				// (cell is empty in out-storage)
				deltaSize --
			}
		} else {
			// storage increases by one new cell
			// (cell is not found in in-storage but found in out-storage)
			deltaSize ++
		}
		// compute update size
		if (outValue != common.Hash{}) {
			outUpdateSize ++
		}
	}
	for address, inValue := range inUpdateSet {
		if _, found := outUpdateSet[address]; !found {
			// storage shrinks by one cell
			// (The cell does not exist for an address in in-storage)
			if (inValue != common.Hash{}) {
				deltaSize --
			}
		}
		if (inValue != common.Hash{}) {
			inUpdateSize ++
		}
	}
	return deltaSize, inUpdateSize, outUpdateSize
}

func main() {
        fmt.Println("metric: start.")

	flag.Parse()
        first := uint64(*firstBlock)
        last := uint64(*lastBlock)

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
			timestamp := substate.Env.Timestamp
			for wallet, outputAccount := range substate.OutputAlloc {
				var ( deltaSize int64
				      inUpdateSize uint64
				      outUpdateSize uint64 )
				if inputAccount, found := substate.InputAlloc[wallet]; found {
					deltaSize, inUpdateSize, outUpdateSize = computeStorageSizes(inputAccount.Storage, outputAccount.Storage)
				} else {
					deltaSize, inUpdateSize, outUpdateSize = computeStorageSizes(map[common.Hash]common.Hash{}, outputAccount.Storage)
				}
				fmt.Printf("metric: data %v %v %v %v %v %v %v\n",block,timestamp,tx,strings.ToLower(wallet.Hex()),deltaSize * 32, inUpdateSize * 32, outUpdateSize * 32)
			}
			for wallet, inputAccount := range substate.InputAlloc {
				var ( deltaSize int64
				      inUpdateSize uint64
				      outUpdateSize uint64 )
				if _, found := substate.OutputAlloc[wallet]; !found {
					deltaSize, inUpdateSize, outUpdateSize = computeStorageSizes(inputAccount.Storage, map[common.Hash]common.Hash{})
					fmt.Printf("metric: data %v %v %v %v %v %v %v\n",block,timestamp,tx,strings.ToLower(wallet.Hex()),deltaSize * 32, inUpdateSize * 32, outUpdateSize * 32)
				}
			}
		}
	}
        //research.CloseSubstateDB()
        fmt.Println("metric: end.")
}

