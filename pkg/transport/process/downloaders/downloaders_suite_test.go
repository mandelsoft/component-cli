// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package downloaders_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"
	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-cli/ociclient/oci"
	"github.com/gardener/component-cli/ociclient/test/envtest"
	"github.com/gardener/component-cli/pkg/testutils"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Downloaders Test Suite")
}

var (
	testenv              *envtest.Environment
	ociClient            ociclient.Client
	ociCache             cache.Cache
	keyring              *credentials.GeneralOciKeyring
	testComponent        cdv2.ComponentDescriptor
	localOciBlobResIndex = 0
	localOciBlobData     = []byte("Hello World")
	ociArtifactResIndex  = 1
	expectedOciArtifact  oci.Artifact
)

var _ = BeforeSuite(func() {
	testenv = envtest.New(envtest.Options{
		RegistryBinaryPath: filepath.Join("../../../../", envtest.DefaultRegistryBinaryPath),
		Stdout:             GinkgoWriter,
		Stderr:             GinkgoWriter,
	})
	Expect(testenv.Start(context.Background())).To(Succeed())

	keyring = credentials.New()
	Expect(keyring.AddAuthConfig(testenv.Addr, credentials.AuthConfig{
		Username: testenv.BasicAuth.Username,
		Password: testenv.BasicAuth.Password,
	})).To(Succeed())
	ociCache = cache.NewInMemoryCache()
	var err error
	ociClient, err = ociclient.NewClient(logr.Discard(), ociclient.WithKeyring(keyring), ociclient.WithCache(ociCache))
	Expect(err).ToNot(HaveOccurred())

	uploadTestComponent()
}, 60)

var _ = AfterSuite(func() {
	Expect(testenv.Close()).To(Succeed())
})

func uploadTestComponent() {
	dgst := digest.FromBytes(localOciBlobData)

	fs := memoryfs.New()
	Expect(fs.Mkdir(ctf.BlobsDirectoryName, os.ModePerm)).To(Succeed())

	blobfile, err := fs.Create(ctf.BlobPath(dgst.String()))
	Expect(err).ToNot(HaveOccurred())

	_, err = blobfile.Write(localOciBlobData)
	Expect(err).ToNot(HaveOccurred())

	Expect(blobfile.Close()).To(Succeed())

	ctx := context.TODO()

	localOciBlobAcc, err := cdv2.NewUnstructured(
		cdv2.NewLocalFilesystemBlobAccess(
			dgst.String(),
			"text/plain",
		),
	)
	Expect(err).ToNot(HaveOccurred())

	localOciBlobRes := cdv2.Resource{
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Name:    "local-oci-blob-res",
			Version: "0.1.0",
			Type:    "plain-text",
		},
		Relation: cdv2.LocalRelation,
		Access:   &localOciBlobAcc,
	}

	ociArtifactRef := testenv.Addr + "/test/downloaders/image:0.1.0"

	m, d := testutils.UploadTestManifest(ctx, ociClient, ociArtifactRef)
	a, err := oci.NewManifestArtifact(&oci.Manifest{
		Descriptor: d,
		Data:       m,
	})
	Expect(err).ToNot(HaveOccurred())
	expectedOciArtifact = *a

	ociArtifactAcc, err := cdv2.NewUnstructured(
		cdv2.NewOCIRegistryAccess(
			ociArtifactRef,
		),
	)
	Expect(err).ToNot(HaveOccurred())

	ociArtifactRes := cdv2.Resource{
		IdentityObjectMeta: cdv2.IdentityObjectMeta{
			Name:    "oci-artifact-res",
			Version: "0.1.0",
			Type:    cdv2.OCIImageType,
		},
		Relation: cdv2.LocalRelation,
		Access:   &ociArtifactAcc,
	}

	ociRepo := cdv2.NewOCIRegistryRepository(testenv.Addr+"/test/downloaders", "")
	repoCtx, err := cdv2.NewUnstructured(
		ociRepo,
	)
	Expect(err).ToNot(HaveOccurred())

	localCd := cdv2.ComponentDescriptor{
		ComponentSpec: cdv2.ComponentSpec{
			ObjectMeta: cdv2.ObjectMeta{
				Name:    "github.com/component-cli/test-component",
				Version: "0.1.0",
			},
			Provider: cdv2.InternalProvider,
			RepositoryContexts: []*cdv2.UnstructuredTypedObject{
				&repoCtx,
			},
			Resources: []cdv2.Resource{
				localOciBlobRes,
				ociArtifactRes,
			},
		},
	}

	manifest, err := cdoci.NewManifestBuilder(ociCache, ctf.NewComponentArchive(&localCd, fs)).Build(ctx)
	Expect(err).ToNot(HaveOccurred())

	ociRef, err := cdoci.OCIRef(*ociRepo, localCd.Name, localCd.Version)
	Expect(err).ToNot(HaveOccurred())

	Expect(ociClient.PushManifest(ctx, ociRef, manifest)).To(Succeed())

	cdresolver := cdoci.NewResolver(ociClient)
	actualCd, err := cdresolver.Resolve(ctx, ociRepo, localCd.Name, localCd.Version)
	Expect(err).ToNot(HaveOccurred())

	testComponent = *actualCd
}
