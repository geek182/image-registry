package server

import (
	"reflect"
	"testing"

	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"

	"github.com/openshift/image-registry/pkg/imagestream"
	imageapi "github.com/openshift/image-registry/pkg/origin-common/image/apis/image"

	registryclient "github.com/openshift/image-registry/pkg/dockerregistry/server/client"
	"github.com/openshift/image-registry/pkg/testutil"
)

func TestTagGet(t *testing.T) {
	namespace := "user"
	repo := "app"
	tag := "latest"

	backgroundCtx := context.Background()
	backgroundCtx = testutil.WithTestLogger(backgroundCtx, t)

	fos, imageClient := testutil.NewFakeOpenShiftWithClient(backgroundCtx)
	testImage := testutil.AddRandomImage(t, fos, namespace, repo, tag)

	testcases := []struct {
		title                 string
		tagName               string
		tagValue              distribution.Descriptor
		expectedError         bool
		expectedNotFoundError bool
		pullthrough           bool
		imageManaged          bool
	}{
		{
			title:        "get valid tag from managed image",
			tagName:      tag,
			tagValue:     distribution.Descriptor{Digest: digest.Digest(testImage.Name)},
			pullthrough:  true,
			imageManaged: true,
		},
		{
			title:        "get valid tag from managed image without pullthrough",
			tagName:      tag,
			tagValue:     distribution.Descriptor{Digest: digest.Digest(testImage.Name)},
			pullthrough:  false,
			imageManaged: true,
		},
		{
			title:                 "get valid tag from unmanaged image without pullthrough",
			tagName:               tag,
			pullthrough:           false,
			imageManaged:          false,
			expectedNotFoundError: true,
		},
		{
			title:                 "get missing tag",
			tagName:               tag + "-no-found",
			pullthrough:           true,
			imageManaged:          true,
			expectedError:         true,
			expectedNotFoundError: true,
		},
	}

	for _, tc := range testcases {
		if tc.imageManaged {
			testImage.Annotations[imageapi.ManagedByOpenShiftAnnotation] = "true"
		} else {
			testImage.Annotations[imageapi.ManagedByOpenShiftAnnotation] = "false"
		}

		imageStream := imagestream.New(backgroundCtx, namespace, repo, registryclient.NewFakeRegistryAPIClient(nil, imageClient))

		ts := &tagService{
			TagService:         newTestTagService(nil),
			imageStream:        imageStream,
			pullthroughEnabled: tc.pullthrough,
		}

		resultDesc, err := ts.Get(backgroundCtx, tc.tagName)

		switch err.(type) {
		case distribution.ErrTagUnknown:
			if !tc.expectedNotFoundError {
				t.Fatalf("[%s] unexpected error: %#+v", tc.title, err)
			}
		case nil:
			if tc.expectedError || tc.expectedNotFoundError {
				t.Fatalf("[%s] unexpected successful response", tc.title)
			}
		default:
			if tc.expectedError {
				break
			}
			t.Fatalf("[%s] unexpected error: %#+v", tc.title, err)
		}

		if resultDesc.Digest != tc.tagValue.Digest {
			t.Fatalf("[%s] unexpected result returned", tc.title)
		}
	}
}

func TestTagGetWithoutImageStream(t *testing.T) {
	namespace := "user"
	repo := "app"
	tag := "latest"

	ctx := context.Background()
	ctx = testutil.WithTestLogger(ctx, t)

	_, imageClient := testutil.NewFakeOpenShiftWithClient(ctx)

	imageStream := imagestream.New(ctx, namespace, repo, registryclient.NewFakeRegistryAPIClient(nil, imageClient))

	ts := &tagService{
		TagService:  newTestTagService(nil),
		imageStream: imageStream,
	}

	_, err := ts.Get(ctx, tag)
	if err == nil {
		t.Fatalf("error expected")
	}

	_, ok := err.(distribution.ErrRepositoryUnknown)
	if !ok {
		t.Fatalf("unexpected error: %#+v", err)
	}
}

func TestTagGetAll(t *testing.T) {
	namespace := "user"
	repo := "app"
	tag := "latest"

	ctx := context.Background()
	ctx = testutil.WithTestLogger(ctx, t)

	fos, imageClient := testutil.NewFakeOpenShiftWithClient(ctx)
	testImage := testutil.AddRandomImage(t, fos, namespace, repo, tag)

	testcases := []struct {
		title         string
		expectResult  []string
		expectedError bool
		pullthrough   bool
		imageManaged  bool
	}{
		{
			title:        "get all tags with pullthrough",
			expectResult: []string{tag},
			pullthrough:  true,
			imageManaged: true,
		},
		{
			title:        "get all tags without pullthrough",
			expectResult: []string{tag},
			imageManaged: true,
		},
		{
			title:        "get all tags from unmanaged image without pullthrough",
			expectResult: []string{},
		},
	}

	for _, tc := range testcases {
		if tc.imageManaged {
			testImage.Annotations[imageapi.ManagedByOpenShiftAnnotation] = "true"
		} else {
			testImage.Annotations[imageapi.ManagedByOpenShiftAnnotation] = "false"
		}

		imageStream := imagestream.New(ctx, namespace, repo, registryclient.NewFakeRegistryAPIClient(nil, imageClient))

		ts := &tagService{
			TagService:         newTestTagService(nil),
			imageStream:        imageStream,
			pullthroughEnabled: tc.pullthrough,
		}

		result, err := ts.All(ctx)

		if err != nil && !tc.expectedError {
			t.Fatalf("[%s] unexpected error: %#+v", tc.title, err)
		}

		if !reflect.DeepEqual(result, tc.expectResult) {
			t.Fatalf("[%s] unexpected result: %#+v", tc.title, result)
		}
	}
}

func TestTagGetAllWithoutImageStream(t *testing.T) {
	namespace := "user"
	repo := "app"

	ctx := context.Background()
	ctx = testutil.WithTestLogger(ctx, t)

	_, imageClient := testutil.NewFakeOpenShiftWithClient(ctx)

	imageStream := imagestream.New(ctx, namespace, repo, registryclient.NewFakeRegistryAPIClient(nil, imageClient))

	ts := &tagService{
		TagService:  newTestTagService(nil),
		imageStream: imageStream,
	}

	_, err := ts.All(ctx)
	if err == nil {
		t.Fatalf("error expected")
	}

	_, ok := err.(distribution.ErrRepositoryUnknown)
	if !ok {
		t.Fatalf("unexpected error: %#+v", err)
	}
}

func TestTagLookup(t *testing.T) {
	namespace := "user"
	repo := "app"
	tag := "latest"

	ctx := context.Background()
	ctx = testutil.WithTestLogger(ctx, t)

	fos, imageClient := testutil.NewFakeOpenShiftWithClient(ctx)
	testImage := testutil.AddRandomImage(t, fos, namespace, repo, tag)

	testcases := []struct {
		title         string
		tagValue      distribution.Descriptor
		expectResult  []string
		expectedError bool
		pullthrough   bool
		imageManaged  bool
	}{
		{
			title:        "lookup tags with pullthrough",
			tagValue:     distribution.Descriptor{Digest: digest.Digest(testImage.Name)},
			expectResult: []string{tag},
			pullthrough:  true,
			imageManaged: true,
		},
		{
			title:        "lookup tags without pullthrough",
			tagValue:     distribution.Descriptor{Digest: digest.Digest(testImage.Name)},
			expectResult: []string{tag},
			imageManaged: true,
		},
		{
			title:        "lookup tags by missing digest",
			tagValue:     distribution.Descriptor{Digest: digest.Digest(etcdDigest)},
			expectResult: []string{},
			pullthrough:  true,
			imageManaged: true,
		},
		{
			title:        "lookup tags in unmanaged images without pullthrough",
			tagValue:     distribution.Descriptor{Digest: digest.Digest(testImage.Name)},
			expectResult: []string{},
		},
	}

	for _, tc := range testcases {
		if tc.imageManaged {
			testImage.Annotations[imageapi.ManagedByOpenShiftAnnotation] = "true"
		} else {
			testImage.Annotations[imageapi.ManagedByOpenShiftAnnotation] = "false"
		}

		imageStream := imagestream.New(ctx, namespace, repo, registryclient.NewFakeRegistryAPIClient(nil, imageClient))

		ts := &tagService{
			TagService:         newTestTagService(nil),
			imageStream:        imageStream,
			pullthroughEnabled: tc.pullthrough,
		}

		result, err := ts.Lookup(ctx, tc.tagValue)

		if err != nil {
			if !tc.expectedError {
				t.Fatalf("[%s] unexpected error: %#+v", tc.title, err)
			}
			continue
		} else {
			if tc.expectedError {
				t.Fatalf("[%s] error expected", tc.title)
			}
		}

		if !reflect.DeepEqual(result, tc.expectResult) {
			t.Fatalf("[%s] unexpected result: %#+v", tc.title, result)
		}
	}
}

func TestTagLookupWithoutImageStream(t *testing.T) {
	namespace := "user"
	repo := "app"
	tag := "latest"

	ctx := context.Background()
	ctx = testutil.WithTestLogger(ctx, t)

	fos, imageClient := testutil.NewFakeOpenShiftWithClient(ctx)
	anotherImage := testutil.AddRandomImage(t, fos, namespace, repo+"-another", tag)

	imageStream := imagestream.New(ctx, namespace, repo, registryclient.NewFakeRegistryAPIClient(nil, imageClient))

	ts := &tagService{
		TagService:  newTestTagService(nil),
		imageStream: imageStream,
	}

	_, err := ts.Lookup(ctx, distribution.Descriptor{
		Digest: digest.Digest(anotherImage.Name),
	})
	if err == nil {
		t.Fatalf("error expected")
	}

	_, ok := err.(distribution.ErrRepositoryUnknown)
	if !ok {
		t.Fatalf("unexpected error: %#+v", err)
	}
}

type testTagService struct {
	data  map[string]distribution.Descriptor
	calls map[string]int
}

func newTestTagService(data map[string]distribution.Descriptor) *testTagService {
	b := make(map[string]distribution.Descriptor)
	for d, content := range data {
		b[d] = content
	}
	return &testTagService{
		data:  b,
		calls: make(map[string]int),
	}
}

func (t *testTagService) Get(ctx context.Context, tag string) (distribution.Descriptor, error) {
	t.calls["Get"]++
	desc, exists := t.data[tag]
	if !exists {
		return distribution.Descriptor{}, distribution.ErrTagUnknown{Tag: tag}
	}
	return desc, nil
}

func (t *testTagService) Tag(ctx context.Context, tag string, desc distribution.Descriptor) error {
	t.calls["Tag"]++
	t.data[tag] = desc
	return nil
}

func (t *testTagService) Untag(ctx context.Context, tag string) error {
	t.calls["Untag"]++
	_, exists := t.data[tag]
	if !exists {
		return distribution.ErrTagUnknown{Tag: tag}
	}
	delete(t.data, tag)
	return nil
}

func (t *testTagService) All(ctx context.Context) (tags []string, err error) {
	t.calls["All"]++
	for tag := range t.data {
		tags = append(tags, tag)
	}
	return
}

func (t *testTagService) Lookup(ctx context.Context, desc distribution.Descriptor) (tags []string, err error) {
	t.calls["Lookup"]++
	for tag := range t.data {
		if t.data[tag].Digest == desc.Digest {
			tags = append(tags, tag)
		}
	}
	return
}
