package job

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
)

func TestMutateBatchJob_BackoffLimit(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, k8upv1.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).Build()

	newBatchJob := func() *batchv1.Job {
		return &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "job", Namespace: "ns"}}
	}

	check := &k8upv1.Check{ObjectMeta: metav1.ObjectMeta{Name: "check", Namespace: "ns"}}
	restore := &k8upv1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: "ns"}}

	tests := map[string]struct {
		obj         k8upv1.JobObject
		setLimit    func(limit *int32)
		limit       *int32
		wantBackoff *int32
	}{
		"CheckWithoutLimitKeepsKubernetesDefault": {
			obj:      check,
			setLimit: func(l *int32) { check.Spec.BackoffLimit = l },
		},
		"CheckWithLimit": {
			obj:         check,
			setLimit:    func(l *int32) { check.Spec.BackoffLimit = l },
			limit:       ptr.To[int32](1),
			wantBackoff: ptr.To[int32](1),
		},
		"RestoreWithLimit": {
			obj:         restore,
			setLimit:    func(l *int32) { restore.Spec.BackoffLimit = l },
			limit:       ptr.To[int32](2),
			wantBackoff: ptr.To[int32](2),
		},
		"RestoreWithZeroLimitDisablesRetries": {
			obj:         restore,
			setLimit:    func(l *int32) { restore.Spec.BackoffLimit = l },
			limit:       ptr.To[int32](0),
			wantBackoff: ptr.To[int32](0),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.setLimit(tc.limit)
			batchJob := newBatchJob()

			err := MutateBatchJob(context.Background(), batchJob, tc.obj, NewConfig(c, tc.obj, "s3:endpoint/bucket"), c)
			require.NoError(t, err)

			assert.Equal(t, tc.wantBackoff, batchJob.Spec.BackoffLimit, "job backoff limit")
			if tc.limit != nil {
				assert.NotSame(t, tc.limit, batchJob.Spec.BackoffLimit, "job must get its own copy")
			}
		})
	}
}
