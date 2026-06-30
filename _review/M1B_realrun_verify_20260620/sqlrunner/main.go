// Throwaway SQL runner for M1-B BC verification (zonky PG ships no psql client).
// Usage: sqlrunner <exec|query>   ; reads SQL from stdin.
//   exec  : runs a (possibly multi-statement) script, prints "OK".
//   query : runs a query, prints column header + rows as TSV (NULL for nulls).
// DSN via M1B_DSN env or default local throwaway cluster. ¥0, local-only.
package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sqlrunner <exec|query>  (SQL on stdin)")
		os.Exit(2)
	}
	mode := os.Args[1]
	dsn := os.Getenv("M1B_DSN")
	if dsn == "" {
		dsn = "host=127.0.0.1 port=5433 user=postgres password=postgres dbname=postgres sslmode=disable"
	}
	raw, _ := io.ReadAll(os.Stdin)
	sqlText := strings.TrimSpace(string(raw))
	if sqlText == "" {
		fmt.Fprintln(os.Stderr, "no SQL on stdin")
		os.Exit(2)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	defer db.Close()

	switch mode {
	case "exec":
		// lib/pq sends a paramless query via the simple protocol -> multi-statement OK.
		if _, err := db.Exec(sqlText); err != nil {
			fmt.Fprintln(os.Stderr, "exec error:", err)
			os.Exit(1)
		}
		fmt.Println("OK")
	case "query":
		rows, err := db.Query(sqlText)
		if err != nil {
			fmt.Fprintln(os.Stderr, "query error:", err)
			os.Exit(1)
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		fmt.Println(strings.Join(cols, "\t"))
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		for rows.Next() {
			if err := rows.Scan(ptrs...); err != nil {
				fmt.Fprintln(os.Stderr, "scan error:", err)
				os.Exit(1)
			}
			out := make([]string, len(cols))
			for i, v := range vals {
				switch t := v.(type) {
				case nil:
					out[i] = "NULL"
				case []byte:
					out[i] = string(t)
				default:
					out[i] = fmt.Sprintf("%v", t)
				}
			}
			fmt.Println(strings.Join(out, "\t"))
		}
		if err := rows.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "rows error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", mode)
		os.Exit(2)
	}
}
