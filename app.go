package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
	cryptoSsh "golang.org/x/crypto/ssh"
)

type App struct {
	*ssh.Server

	progs map[string]*tea.Program

	playerToRoom map[string]*Room
}

func NewApp() *App {
	app := App{
		progs:        make(map[string]*tea.Program),
		playerToRoom: make(map[string]*Room),
	}

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithPublicKeyAuth(func(_ ssh.Context, key ssh.PublicKey) bool {
			return true
		}),
		wish.WithMiddleware(
			bubbletea.MiddlewareWithProgramHandler(app.ProgramHandler, termenv.ANSI256),
			logging.Middleware(),
		),
	)

	if err != nil {
		log.Fatal("Could not start server", "error", err)
	}

	app.Server = s
	return &app
}

func (app *App) Start() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err := app.Server.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := app.Server.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

// TODO: auth?
func (app *App) Send(player string, msg tea.Msg) {
	if room, exists := app.playerToRoom[player]; exists {
		for _, p := range room.players {
			app.progs[p].Send(msg)
		}
	} else {
		// TODO: error handling
	}
}

func (app *App) ProgramHandler(sess ssh.Session) *tea.Program {
	_, _, active := sess.Pty()
	if !active {
		wish.Fatalln(sess, "no active terminal, skipping")
		return nil
	}

	user := cryptoSsh.FingerprintSHA256(sess.PublicKey())

	if _, exist := app.progs[user]; exist {
		wish.Fatalln(sess, "user has another session")
	}

	m := newModel(keymap{
		up:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
		down:   key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
		left:   key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left")),
		right:  key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right")),
		choose: key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "(un)select")),
	})

	prog := tea.NewProgram(m, append(bubbletea.MakeOptions(sess), tea.WithAltScreen())...)
	app.progs[user] = prog

	return prog
}
