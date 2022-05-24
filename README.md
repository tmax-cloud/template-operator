# Template-operator

> Template-operator is Kubernetes-native deployment service. With setted templates, you can deploy k8s applications by creating template instances.
> Supports both namespaced scope templates and cluster scope cluster templates.  

![image](https://raw.githubusercontent.com/tmax-cloud/template-operator/master/docs/template.JPG)

## prerequisite Install
- kubernetes

## Build
- [Image-build](#image-build)
- [Image-push](#image-push)

---

#### Image-build
> 오퍼레이터 이미지를 빌드 합니다.
- make docker-build IMG={YOUR_REPOSITORY}/{IMAGE_NAME}:{TAG}
- 예시: make docker-build IMG=192.168.6.122:5000/template-operator:0.0.1

#### Image-push
> 이미지 레지스트리에 이미지를 푸쉬 합니다.
- make docker-push IMG={YOUR_REPOSITORY}/{IMAGE_NAME}:{TAG}
- 예시: make docker-push IMG=192.168.6.122:5000/template-operator:0.0.1

## Install Template Operator

- [CRD](#crd)
- [Namespace](#namespace)
- [RBAC](#RBAC)
- [Deployment](#deployment)
- [Test](#test)

---

#### crd
> crd를 생성 합니다.
- kubectl apply -f tmax.io_templates.yaml ([파일](./config/crd/bases/tmax.io_templates.yaml))
- kubectl apply -f tmax.io_clustertemplates.yaml ([파일](./config/crd/bases/tmax.io_clustertemplates.yaml))
- kubectl apply -f tmax.io_templateinstances.yaml ([파일](./config/crd/bases/tmax.io_templateinstances.yaml))
- kubectl apply -f tmax.io_catalogserviceclaims.yaml ([파일](./config/crd/bases/tmax.io_catalogserviceclaims.yaml))

---

#### Namespace
> 오퍼레이터를 위한 네임스페이스를 생성 합니다.
- kubectl create namespace template

---

#### RBAC
> 서비스어카운트를 생성 합니다.
> 서비스어카운트를 위한 Role을 생성 합니다.
> RoleBinding을 생성 합니다.
- kubectl apply -f deploy_rbac.yaml ([파일](./config/rbac/deploy_admin_rbac.yaml))

---

#### Deployment
> Template Operator를 생성 합니다.
>> 단, deploy_manager 내부의 image 경로는 사용자 환경에 맞게 수정 해야 합니다.
- kubectl apply -f deploy_manager.yaml ([파일](./config/manager/deploy_manager.yaml))

---

#### Test
> 테스트는 config/samples 디렉토리를 참고하여 하시면 됩니다.
- kubectl apply -f example.yaml

---

## Delete Template Operator

- [Delete-resource](#Delete-resource)
- [Delete-deployment](#Delete-deployment)
- [Delete-rbac](#Delete-rbac)
- [Delete-crd](#Delete-crd)
- [Delete-namespace](#Delete-namespace)

---

#### Delete-resource
> Template 관련 리소스를 모두 삭제 합니다.
- kubectl delete templateinstance --all --all-namespaces
- kubectl delete catalogserviceclaim --all --all-namespaces
- kubectl delete clustertemplate --all --all-namespaces
- kubectl delete template --all --all-namespaces

---

#### Delete-deployment
> Template operator를 삭제 합니다.
- kubectl delete -f deploy_manager.yaml ([파일](./config/manager/deploy_manager.yaml))

---

#### Delete-rbac
> template 관련 rbac을 삭제 합니다.
- kubectl delete -f deploy_rbac.yaml ([파일](./config/rbac/deploy_admin_rbac.yaml))

---

#### Delete-crd
> template 관련 CRD를 삭제 합니다.
- kubectl delete -f tmax.io_templates.yaml ([파일](./config/crd/bases/tmax.io_templates.yaml))
- kubectl delete -f tmax.io_clustertemplates.yaml ([파일](./config/crd/bases/tmax.io_clustertemplates.yaml))
- kubectl delete -f tmax.io_templateinstances.yaml ([파일](./config/crd/bases/tmax.io_templateinstances.yaml))
- kubectl delete -f tmax.io_catalogserviceclaims.yaml ([파일](./config/crd/bases/tmax.io_catalogserviceclaims.yaml))

---

#### Delete-namespace
> template namespace를 삭제 합니다.
- kubectl delete namespace template 

---

#### Changes
> hypercloud 5.0 변경 사항 입니다.
1. ClusterTemplate 추가
    - default 네임스페이스에 만들어서 사용 하던 사용자 공통 template을 cluster-scope의 ClusterTemplate을 통해 사용
2. CatalogServiceClaim 변경
    - Approve 상태 추가 (Approve 후, 성공적으로 ClusterTemplate 만들어지면 Success 상태로 변경)
    - 승인 후, Template이 아닌 ClusterTemplate이 생성
3. TemplateInstance 사용법 변경
    - ClusterTemplate을 기반으로 instance 생성 할 경우 [파일](./config/samples/cluster-example-template-instance.yaml)과 같이 사용
    - Template 기반으로 instance 생성 할 시에는 기존과 동일하게 사용 가능
4. template-operator 와 template-service-broker가 독립적으로 동작 할 수 있도록 로직 분리 
5. Gitops option 기능 추가
    - Source Git 접속 시 필요한 credential은 secret으로 Template Instance 생성 할 Namespace에 먼저 생성 (User ID / Access token). 예시) [파일](./config/samples/secret.yaml)
    - Template Instance 생성 시 annotations field에 "gitops: enable" 추가. 예시) [파일](./config/samples/gitops-example-instance.yaml)
    - Template Instance Spec에 Template manifests push할 Source Git repo와 path 입력. 예시) [파일](./config/samples/gitops-example-instance.yaml)
    
