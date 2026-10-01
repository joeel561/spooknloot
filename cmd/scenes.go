package main

import (
	"spooknloot/pkg/lobby"
	"spooknloot/pkg/netcode"
	"spooknloot/pkg/player"
	"spooknloot/pkg/ui"
)

// drawMenuScreens draws the title, join and lobby screens on top of the
// world and applies whatever the player clicked.
func drawMenuScreens() {
	switch currentScene {
	case sceneTitle:
		switch ui.DrawTitle(&nameInput, &selectedClass, statusMessage) {
		case ui.TitleSingleplayer:
			statusMessage = ""
			startGame()
		case ui.TitleHost:
			l, err := lobby.NewHost(nameInput.Text, selectedClass, netcode.DefaultGamePort)
			if err != nil {
				statusMessage = err.Error()
				return
			}
			statusMessage = ""
			gameLobby = l
			currentScene = sceneLobby
		case ui.TitleJoin:
			statusMessage = ""
			if b, err := netcode.StartBrowser(); err != nil {
				statusMessage = "LAN search unavailable, join by IP instead"
			} else {
				lanBrowser = b
			}
			currentScene = sceneJoin
		case ui.TitleQuit:
			running = false
		}

	case sceneJoin:
		var games []netcode.FoundGame
		if lanBrowser != nil {
			games = lanBrowser.Games()
		}
		action, addr := ui.DrawJoin(games, &addressInput, statusMessage)
		switch action {
		case ui.JoinConnect:
			statusMessage = ""
			stopLANBrowser()
			gameLobby = lobby.Join(addr, nameInput.Text, selectedClass)
			currentScene = sceneLobby
		case ui.JoinBack:
			statusMessage = ""
			stopLANBrowser()
			currentScene = sceneTitle
		}

	case sceneLobby:
		switch ui.DrawLobby(gameLobby) {
		case ui.LobbyToggleReady:
			me, _ := gameLobby.LocalPlayer()
			gameLobby.SetReady(!me.Ready)
		case ui.LobbyStart:
			gameLobby.Start()
		case ui.LobbyLeave:
			returnToTitle("")
		}
	}
}

// updateLobby keeps the network running and reacts to the lobby starting
// the game or closing (e.g. host left).
func updateLobby() {
	if gameLobby == nil {
		return
	}
	if currentScene == scenePlaying {
		gameLobby.SetLocalState(localPlayerState())
	}
	gameLobby.Update()
	if gameLobby.State == lobby.StateClosed {
		msg := ""
		if gameLobby.Err != nil {
			msg = gameLobby.Err.Error()
		}
		returnToTitle(msg)
		return
	}
	if gameLobby.TakeStarted() {
		startGame()
	}
}

func startGame() {
	applyClass(localClass())
	resetGameState()
	startSession()
	currentScene = scenePlaying
}

func returnToTitle(message string) {
	if gameLobby != nil {
		gameLobby.Leave()
		gameLobby = nil
	}
	stopLANBrowser()
	resetGameState()
	if menuPausedMusic {
		resumeCurrentMusic()
		menuPausedMusic = false
	}
	statusMessage = message
	currentScene = sceneTitle
}

func resetGameState() {
	session = nil
	enterWorld()
	bossWinOpen = false
	inventoryOpen = false
	toasts = nil
	menuOpen = false
	player.ResetPlayer()
}

func stopLANBrowser() {
	if lanBrowser != nil {
		lanBrowser.Close()
		lanBrowser = nil
	}
}
