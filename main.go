package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
)


type Block struct {
	Index     int
	Timestamp string
	Content   string
	Hash      string
	PrevHash  string
}

var Blockchain []Block
var bcServer chan []Block
var mutex = &sync.Mutex{}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	bcServer = make(chan []Block)

	t := time.Now()
	genesisBlock := Block{0, t.String(), "666", "", ""}
	spew.Dump(genesisBlock)
	Blockchain = append(Blockchain, genesisBlock)

	// 开启对9000端口的监听
	server, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConn(conn)
	}

}

func handleConn(conn net.Conn) {

	defer conn.Close()

	io.WriteString(conn, "请输入任意内容:")

	scanner := bufio.NewScanner(conn)


	go func() {
		for scanner.Scan() {
			content := scanner.Text()

			newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], string(content))
			if err != nil {
				log.Println(err)
				continue
			}
			//取最长的链为有效链
			if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
				newBlockchain := append(Blockchain, newBlock)
				replaceChain(newBlockchain)
			}

			bcServer <- Blockchain
			io.WriteString(conn, "\n请输入任意内容:")
			
		}
	}()

	// 定时器进行广播
	go func() {
		for {
			time.Sleep(30 * time.Second)
			mutex.Lock()
			output, err := json.Marshal(Blockchain)
			if err != nil {
				log.Fatal(err)
			}
			mutex.Unlock()
			io.WriteString(conn, string(output))
		}
	}()

	for _ = range bcServer {
		spew.Dump(Blockchain)
	}

}

func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

//以最长的链为有效链
func replaceChain(newBlocks []Block) {
	mutex.Lock()
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
	mutex.Unlock()
}

func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.Content) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func generateBlock(oldBlock Block, Content string) (Block, error) {

	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.Content = Content
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
}
