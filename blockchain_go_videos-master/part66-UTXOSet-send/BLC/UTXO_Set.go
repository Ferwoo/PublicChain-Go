package BLC

import (
	"github.com/boltdb/bolt"
	"log"
	"encoding/hex"
	"fmt"
)



const utxoTableName  = "utxoTableName"

type UTXOSet struct {
	Blockchain *Blockchain
}

// 重置数据库表
func (utxoSet *UTXOSet) ResetUTXOSet()  {

	err := utxoSet.Blockchain.DB.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(utxoTableName))

		if b != nil {


			err := tx.DeleteBucket([]byte(utxoTableName))

			if err!= nil {
				log.Panic(err)
			}

		}

		b ,_ = tx.CreateBucket([]byte(utxoTableName))
		if b != nil {

			//[string]*TXOutputs
			txOutputsMap := utxoSet.Blockchain.FindUTXOMap()


			for keyHash,outs := range txOutputsMap {

				txHash,_ := hex.DecodeString(keyHash)

				b.Put(txHash,outs.Serialize())

			}
		}

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

}

func (utxoSet *UTXOSet) findUTXOForAddress(address string) []*UTXO{


	var utxos []*UTXO

	utxoSet.Blockchain.DB.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(utxoTableName))

		// 游标
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {

			txOutputs := DeserializeTXOutputs(v)

			for _,utxo := range txOutputs.UTXOS  {

				if utxo.Output.UnLockScriptPubKeyWithAddress(address) {
					utxos = append(utxos,utxo)
				}
			}
		}

		return nil
	})

	return utxos
}




func (utxoSet *UTXOSet) GetBalance(address string) int64 {

	UTXOS := utxoSet.findUTXOForAddress(address)

	var amount int64

	for _,utxo := range UTXOS  {
		amount += utxo.Output.Value
	}

	return amount
}


// 返回要凑多少钱，对应TXOutput的TX的Hash和index
func (utxoSet *UTXOSet) FindUnPackageSpendableUTXOS(from string, txs []*Transaction) []*UTXO {

	var unUTXOs []*UTXO

	spentTXOutputs := make(map[string][]int)

	//{hash:[0]}

	for _,tx := range txs {

		if tx.IsCoinbaseTransaction() == false {
			for _, in := range tx.Vins {
				//是否能够解锁
				publicKeyHash := Base58Decode([]byte(from))

				ripemd160Hash := publicKeyHash[1:len(publicKeyHash) - 4]
				if in.UnLockRipemd160Hash(ripemd160Hash) {

					key := hex.EncodeToString(in.TxHash)

					spentTXOutputs[key] = append(spentTXOutputs[key], in.Vout)
				}

			}
		}
	}


	for _,tx := range txs {

	Work1:
		for index,out := range tx.Vouts {

			if out.UnLockScriptPubKeyWithAddress(from) {
				fmt.Println("看看是否是俊诚...")
				fmt.Println(from)

				fmt.Println(spentTXOutputs)

				if len(spentTXOutputs) == 0 {
					utxo := &UTXO{tx.TxHash, index, out}
					unUTXOs = append(unUTXOs, utxo)
				} else {
					for hash,indexArray := range spentTXOutputs {

						txHashStr := hex.EncodeToString(tx.TxHash)

						if hash == txHashStr {

							var isUnSpentUTXO bool

							for _,outIndex := range indexArray {

								if index == outIndex {
									isUnSpentUTXO = true
									continue Work1
								}

								if isUnSpentUTXO == false {
									utxo := &UTXO{tx.TxHash, index, out}
									unUTXOs = append(unUTXOs, utxo)
								}
							}
						} else {
							utxo := &UTXO{tx.TxHash, index, out}
							unUTXOs = append(unUTXOs, utxo)
						}
					}
				}

			}

		}

	}

	return unUTXOs

}

func (utxoSet *UTXOSet) FindSpendableUTXOS(from string,amount int64,txs []*Transaction) (int64,map[string][]int)  {

	unPackageUTXOS := utxoSet.FindUnPackageSpendableUTXOS(from,txs)

	spentableUTXO := make(map[string][]int)

	var money int64 = 0

	for _,UTXO := range unPackageUTXOS {

		money += UTXO.Output.Value;
		txHash := hex.EncodeToString(UTXO.TxHash)
		spentableUTXO[txHash] = append(spentableUTXO[txHash],UTXO.Index)
		if money >= amount{
			return  money,spentableUTXO
		}
	}


	// 钱还不够
	utxoSet.Blockchain.DB.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(utxoTableName))

		if b != nil {

			c := b.Cursor()
			UTXOBREAK:
			for k, v := c.First(); k != nil; k, v = c.Next() {

				txOutputs := DeserializeTXOutputs(v)

				for _,utxo := range txOutputs.UTXOS {

					money += utxo.Output.Value
					txHash := hex.EncodeToString(utxo.TxHash)
					spentableUTXO[txHash] = append(spentableUTXO[txHash],utxo.Index)

					if money >= amount {
						 break UTXOBREAK;
					}
				}
			}

		}

		return nil
	})

	if money < amount{
		log.Panic("余额不足......")
	}


	return  money,spentableUTXO
}





