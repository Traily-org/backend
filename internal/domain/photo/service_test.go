package photo_test

import (
	"context"
	"testing"

	"github.com/traily-org/server/internal/domain/photo"
)

type fakeRepo struct{ photo.Repository }

func (fakeRepo) Create(_ context.Context, p photo.Photo) (photo.Photo, error) { return p, nil }

// Une photo créée attend toujours la vérification NSFW, même si le client envoie "safe".
func TestService_Create_ForcesNSFWPending(t *testing.T) {
	got, err := photo.NewService(fakeRepo{}).Create(context.Background(), photo.Photo{NSFWStatus: "safe"})
	if err != nil {
		t.Fatal(err)
	}
	if got.NSFWStatus != "pending" || got.Visibility != "public" {
		t.Fatalf("got nsfw=%q visibility=%q", got.NSFWStatus, got.Visibility)
	}
}
