package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"document-mdp/internal/app"
	"document-mdp/internal/config"
	"document-mdp/internal/db"
	"document-mdp/internal/service/auth"
)

func main() {
	var (
		serve     bool
		initUser  bool
		email     string
		password  string
		name      string
		groupName string
	)

	flag.BoolVar(&serve, "serve", false, "run http server")
	flag.BoolVar(&initUser, "init-user", false, "initialize first user (idempotent)")
	flag.StringVar(&email, "email", "", "user email (for --init-user)")
	flag.StringVar(&password, "password", "", "user password (for --init-user)")
	flag.StringVar(&name, "name", "", "user name (for --init-user)")
	flag.StringVar(&groupName, "group", "", "optional group name to create and add user into (for --init-user)")
	flag.Parse()

	if !serve && !initUser {
		flag.Usage()
		os.Exit(2)
	}

	if initUser {
		if email == "" || password == "" || name == "" {
			log.Fatal("--email, --password, --name are required with --init-user")
		}
		if err := initFirstUser(email, password, name, groupName); err != nil {
			log.Fatal(err)
		}
		log.Printf("init-user done: %s", email)
	}

	if serve {
		if err := app.Run(); err != nil {
			log.Fatal(err)
		}
	}
}

func initFirstUser(email, password, name, groupName string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	h, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		_ = h.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := db.Migrate(ctx, h); err != nil {
		return err
	}

	a := auth.NewService(h.Ent, cfg)
	u, created, err := a.EnsureUser(ctx, email, password, name)
	if err != nil {
		return err
	}
	if groupName != "" {
		g, _, err := a.EnsureGroup(ctx, groupName)
		if err != nil {
			return err
		}
		if err := a.AddUserToGroup(ctx, u.ID, g.ID); err != nil {
			return err
		}
	}

	if created {
		log.Printf("created user id=%s", u.ID)
	} else {
		log.Printf("updated user id=%s", u.ID)
	}
	return nil
}
