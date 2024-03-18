package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	roomRepo     RoomRepository
	tableRepo    ArithmeticTableRepository
}

func NewApp() *App {
	app := App{
		progs:        make(map[string]*tea.Program),
		playerToRoom: make(map[string]*Room),
		roomRepo:     NewInMemoryRoomRepository(),
		tableRepo:    NewInMemoryArithmeticTableRepository(),
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.Server.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func (app *App) Send(player string, msg tea.Msg) {
	if room, exists := app.playerToRoom[player]; exists {
		for _, p := range room.players {
			go app.progs[p].Send(msg)
		}
	} else {
		log.Errorf("user not found: %s", player)
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

	m := NewAppModel(user, app)

	// route to room page
	gm := NewRoomPage(30, 80, app.roomRepo)
	m.router.Goto(StaticRoute{Model: gm})

	// listen to connection close
	go func() {
		ctx := sess.Context()
		<-ctx.Done()

		// release user resource
		delete(app.progs, user)

		if r, exists := app.playerToRoom[user]; exists {
			r.RemovePlayer(user)
			if len(r.players) == 0 {
				app.roomRepo.Remove(r.id)
			}
		}

		delete(app.playerToRoom, user)

		app.tableRepo.RemoveByPlayer(user)

		log.Infof("Good bye %s", user)
	}()

	prog := tea.NewProgram(m, append(bubbletea.MakeOptions(sess), tea.WithAltScreen())...)
	app.progs[user] = prog

	return prog
}
