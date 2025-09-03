package plugin

import (
	"context"

	"code.houdinigraphql.com/packages/houdini-core/plugin/documents"
	"code.houdinigraphql.com/packages/houdini-core/plugin/documents/artifacts"
)

func (p *HoudiniCore) Generate(ctx context.Context) error {
	// the first thing to do is generate the artifacts
	err := artifacts.Generate(ctx, p.DB, p.Fs, false)
	if err != nil {
		return err
	}

	// generate the persisted queries document
	err = documents.GeneratePersistentQueries(ctx, p.DB, p.Fs)
	if err != nil {
		return err
	}

	// we're done
	return nil
}
