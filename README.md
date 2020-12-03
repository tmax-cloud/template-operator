# Template-operator

> Template-operator for HyperCloud Service

**Architecture**

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
- [Changes](#changes)

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
- kubectl create namespace {YOUR_NAMESPACE}

---

#### RBAC
> 서비스어카운트를 생성 합니다.
> 서비스어카운트를 위한 Role을 생성 합니다.
> RoleBinding을 생성 합니다.
>> 단, ClusterRoleBinding 내부의 namespace(default)를 {YOUR_NAMESPACE}로 변경해주어야 합니다.
- kubectl apply -f deploy_rbac.yaml -n {YOUR_NAMESPACE} ([파일](./config/rbac/deploy_admin_rbac.yaml))

---

#### Deployment
> Template Operator를 생성 합니다.
>> 단, deploy_manager 내부의 image 경로는 사용자 환경에 맞게 수정 해야 합니다.
- kubectl apply -f deploy_manager.yaml -n {YOUR_NAMESPACE} ([파일](./config/manager/deploy_manager.yaml))

---

#### Test
> 테스트는 config/samples 디렉토리를 참고하여 하시면 됩니다.

- kubectl apply -f example.yaml

---

#### Changes
> hypercloud 4.2 변경 사항 입니다.
1. ClusterTemplate 추가
    - default 네임스페이스에 만들어서 사용 하던 사용자 공통 template을 cluster-scope의 ClusterTemplate을 통해 사용
2. CatalogServiceClaim 변경
    - Approve 상태 추가 (Approve 후, 성공적으로 ClusterTemplate 만들어지면 Success 상태로 변경)
    - 승인 후, Template이 아닌 ClusterTemplate이 생성
3. TemplateInstance 사용법 변경
    - ClusterTemplate을 기반으로 instance 생성 할 경우 [파일](./config/samples/cluster-example-template-instance.yaml)과 같이 사용
    - Template 기반으로 instance 생성 할 시에는 기존과 동일하게 사용 가능
4. template-operator 와 template-service-broker가 독립적으로 동작 할 수 있도록 로직 분리 