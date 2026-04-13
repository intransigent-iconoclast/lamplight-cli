package client

import (
	"context"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
)

type ProviderClient interface {
	RetrieveIndexers(ctx context.Context, provider *entity.Provider) ([]dao.ProviderIndexerDAO, error)
}
