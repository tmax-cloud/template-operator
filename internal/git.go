package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-git/v5/plumbing"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	billy "github.com/go-git/go-billy/v5"
	memfs "github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/object"
	http "github.com/go-git/go-git/v5/plumbing/transport/http"
	memory "github.com/go-git/go-git/v5/storage/memory"
	tmplv1 "github.com/tmax-cloud/template-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// [TODO] : err 처리 log로 바꾸기
var (
	defaultRemoteName = "main"
	storer            *memory.Storage
	fs                billy.Filesystem
)

func PushToGivenRepo(instance *tmplv1.TemplateInstance, obj runtime.RawExtension, c client.Client) error {
	storer = memory.NewStorage()
	fs = memfs.New()

	credential := &corev1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      instance.Spec.Gitops.Secret,
	}, credential); err != nil {
		fmt.Printf("%v", err)
		return err
	}

	// Authentication
	auth := &http.BasicAuth{
		Username: string(credential.Data["username"]),
		Password: string(credential.Data["token"]), // personal access token
	}

	repository := MutateRepoURL(instance.Spec.Gitops.SourceGitRepo)
	repo, err := git.Clone(storer, fs, &git.CloneOptions{
		URL:  repository,
		Auth: auth,

		RemoteName:    defaultRemoteName,
		ReferenceName: plumbing.ReferenceName("refs/heads/main"),
		SingleBranch:  true,
		Tags:          git.NoTags,
	})
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	fmt.Println("Repository cloned")

	w, err := repo.Worktree()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	getKind := make(map[string]interface{})
	if err := json.Unmarshal(obj.Raw, &getKind); err != nil {
		fmt.Println(err, "error occurs while unmarshal")
	}
	objKind := getKind["kind"].(string)

	// Create new file
	path := MutateRepoPath(instance.Spec.Gitops.Path)
	filePath := path + "/" + instance.Name + "_" + objKind + ".yaml"
	newFile, err := fs.Create(filePath)
	if err != nil {
		return err
	}

	yamlRaw, err := yaml.JSONToYAML(obj.Raw)
	if err != nil {
		return err
	}
	newFile.Write(yamlRaw)
	newFile.Close()

	// git add $filePath
	_, err = w.Add(filePath)
	if err != nil {
		return err
	}

	// git commit -m $message
	commitTime := time.Now()
	_, err = w.Commit("Update"+" "+instance.Name+"_"+objKind+".yaml", &git.CommitOptions{
		Author: &object.Signature{
			Email: auth.Username,
			When:  commitTime,
		},
	})
	if err != nil {
		return err
	}

	//Push the code to the remote
	err = repo.Push(&git.PushOptions{
		RemoteName: defaultRemoteName,
		Auth:       auth,
		Prune:      true,
	})
	if err != nil {
		return err
	}

	fmt.Println("Remote updated.", filePath)

	return nil
}

func MutateRepoURL(Repo string) (result string) {
	if ok := strings.HasPrefix(Repo, "https://"); ok {
		return Repo
	}

	result = "https://" + Repo

	return result
}

func MutateRepoPath(Path string) (result string) {
	result = Path
	if ok := strings.HasPrefix(Path, "/"); ok {
		result = Path[1:]
	}

	if ok := strings.HasSuffix(Path, "/"); ok {
		result = result[0 : len(result)-1]
	}

	return result
}
