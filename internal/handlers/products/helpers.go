package products

import (
	"database/sql"
	"strings"
)

func normalizeAliases(in []string) []string {
	if len(in) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))

	for _, a := range in {
		s := strings.TrimSpace(a)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func insertAliases(tx *sql.Tx, productID int64, aliases []string) error {
	if len(aliases) == 0 {
		return nil
	}

	stmt, err := tx.Prepare(`INSERT INTO product_aliases(product_id, alias) VALUES ($1, $2)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, a := range aliases {
		if _, err := stmt.Exec(productID, a); err != nil {
			return err
		}
	}
	return nil
}
