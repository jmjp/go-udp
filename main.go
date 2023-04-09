package main

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

type Message struct {
	ID    int    `json:"id"`
	Data  string `json:"data"`
	Total int    `json:"total"`
	Index int    `json:"index"`
}

type MessageStore struct {
	messages map[int][][]byte
	mutex    sync.Mutex
}

func NewMessageStore() *MessageStore {
	return &MessageStore{
		messages: make(map[int][][]byte),
	}
}

func (store *MessageStore) AddMessage(msg *Message) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if _, ok := store.messages[msg.ID]; !ok {
		store.messages[msg.ID] = make([][]byte, msg.Total)
	}
	store.messages[msg.ID][msg.Index] = []byte(msg.Data)
}

func (store *MessageStore) GetMessage(id int) ([]byte, bool) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	fragments, ok := store.messages[id]
	if !ok {
		return nil, false
	}

	complete := true
	for _, fragment := range fragments {
		if fragment == nil {
			complete = false
			break
		}
	}
	if !complete {
		return nil, false
	}

	// concatenar os fragmentos em uma única mensagem
	message := make([]byte, 0)
	for _, fragment := range fragments {
		message = append(message, fragment...)
	}

	// remover a mensagem da memória
	delete(store.messages, id)

	return message, true
}

func main() {
	store := NewMessageStore()

	// configurar o servidor para receber pacotes UDP
	addr, err := net.ResolveUDPAddr("udp", ":3000")
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Println("Servidor UDP iniciado.")

	// ler e processar cada pacote recebido
	buffer := make([]byte, 65535)
	for {
		n, _, err := conn.ReadFromUDP(buffer)

		if err != nil {
			panic(err)
		}

		// decodificar a mensagem do JSON
		var message Message
		err = json.Unmarshal(buffer[:n], &message)
		if err != nil {
			fmt.Println("Erro ao decodificar a mensagem:", err)
			continue
		}

		// adicionar o fragmento de mensagem à memória
		store.AddMessage(&message)

		// verificar se a mensagem está completa
		completeMsg, ok := store.GetMessage(message.ID)
		if ok {
			fmt.Printf("Mensagem completa recebida (%d bytes): %s\n", len(completeMsg), completeMsg)
		}

		// enviar a confirmação de recebimento da mensagem
		response := "Mensagem recebida com sucesso"
		_, err = conn.WriteToUDP([]byte(response), addr)
		if err != nil {
			fmt.Println("Erro ao enviar resposta:", err)
		}

	}
}
