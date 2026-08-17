package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "os"
    "sort"
    "strings"
    "time"

    _ "github.com/lib/pq"

    repomigrations "github.com/WolcenOn/Supermarket-Prices-API/migrations"
)

const migrationLockID int64 = 784251903

func main() {
    databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
    if databaseURL == "" {
        log.Fatal("DATABASE_URL is required")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    db, err := sql.Open("postgres", databaseURL)
    if err != nil {
        log.Fatalf("open postgres: %v", err)
    }
    defer db.Close()

    conn, err := db.Conn(ctx)
    if err != nil {
        log.Fatalf("acquire postgres connection: %v", err)
    }
    defer conn.Close()

    if err := conn.PingContext(ctx); err != nil {
        log.Fatalf("ping postgres: %v", err)
    }

    if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, migrationLockID); err != nil {
        log.Fatalf("acquire migration lock: %v", err)
    }
    defer func() {
        unlockCtx, unlockCancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer unlockCancel()
        _, _ = conn.ExecContext(unlockCtx, `SELECT pg_advisory_unlock($1)`, migrationLockID)
    }()

    if _, err := conn.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            name TEXT PRIMARY KEY,
            applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )
    `); err != nil {
        log.Fatalf("ensure schema_migrations: %v", err)
    }

    entries, err := repomigrations.Files.ReadDir(".")
    if err != nil {
        log.Fatalf("read embedded migrations: %v", err)
    }

    names := make([]string, 0, len(entries))
    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
            continue
        }
        names = append(names, entry.Name())
    }
    sort.Strings(names)

    applied := 0
    for _, name := range names {
        ok, err := alreadyApplied(ctx, conn, name)
        if err != nil {
            log.Fatal(err)
        }
        if ok {
            log.Printf("skip %s", name)
            continue
        }

        sqlBytes, err := repomigrations.Files.ReadFile(name)
        if err != nil {
            log.Fatalf("read migration %s: %v", name, err)
        }
        if err := applyMigration(ctx, conn, name, string(sqlBytes)); err != nil {
            log.Fatal(err)
        }
        applied++
        log.Printf("applied %s", name)
    }

    log.Printf("migrations complete: %d applied, %d total", applied, len(names))
}

func alreadyApplied(ctx context.Context, conn *sql.Conn, name string) (bool, error) {
    var exists bool
    if err := conn.QueryRowContext(ctx, `
        SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)
    `, name).Scan(&exists); err != nil {
        return false, fmt.Errorf("check migration %s: %w", name, err)
    }
    return exists, nil
}

func applyMigration(ctx context.Context, conn *sql.Conn, name, contents string) error {
    tx, err := conn.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin migration %s: %w", name, err)
    }
    defer tx.Rollback()

    if _, err := tx.ExecContext(ctx, contents); err != nil {
        return fmt.Errorf("execute migration %s: %w", name, err)
    }
    if _, err := tx.ExecContext(ctx, `
        INSERT INTO schema_migrations (name) VALUES ($1)
    `, name); err != nil {
        return fmt.Errorf("record migration %s: %w", name, err)
    }
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit migration %s: %w", name, err)
    }
    return nil
}
