package client

import (
	"context"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/entity"
)

type ProviderClient interface {
	RetrieveIndexers(ctx context.Context, provider *entity.Provider) ([]dao.ProviderIndexerDAO, error)
}
