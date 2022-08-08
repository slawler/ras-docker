# ras-runner

Development space for containerized HEC-RAS simulations

## Getting Started

1. Add a `.env` file to the root level of this directory with the following structure

   ```bash
   AWS_ACCESS_KEY_ID='EXAMPLEXXXXXXID'
   AWS_SECRET_ACCESS_KEY='EXAMPLEXXXXXXXXXXXXEKEY'
   AWS_REGION='us-east-1'
   AWS_BUCKET='bucket-name'
   ```

2. Copy a `payload.yml` to the s3 bucket specified in 1.

3. Add required HEC-RAS files to the s3 bucket (as specified in Step-2)

4. Build and Run Dockerfile.alt (docker-compose included with hot-reload for dev)

---

### Example of Current Manifest (`payload.yml`)

_Note that currently **name**, **format**, and **scheme** are not required by the container but left in as place holders to conform more closely with the payload conceptualized in the [wat-api](https://github.com/USACE/wat-api/blob/768aa27753a1337cc7a3cdfa059f43ca202da433/configs/ras-runner_payload.yml#L1)_

```yaml
payload_id: 17b6ca16-70af-54bd-98d3-0329d69b957b
model:
  name: Muncie
  alternative: .p04
event_index: 0
inputs:
  - id: d1d60c2e-f3b4-436b-9c87-aaaaaaaaaaa1
    filename: Muncie.p04.tmp.hdf
    resource_info:
      store: s3
      root: cloud-wat-dev
      path: runs/test/Muncie/Muncie.p04.tmp.hdf
  - id: cb950b3f-e4ff-4109-936c-aaaaaaaaaaa2
    filename: Muncie.b04
    resource_info:
      store: s3
      root: cloud-wat-dev
      path: runs/test/Muncie/Muncie.b04
  - id: cb950b3f-e4ff-4109-936c-aaaaaaaaaaa3
    filename: Muncie.prj
    resource_info:
      store: s3
      root: cloud-wat-dev
      path: runs/test/Muncie/Muncie.prj
  - id: cb950b3f-e4ff-4109-936c-aaaaaaaaaaa4
    filename: Muncie.x04
    resource_info:
      store: s3
      root: cloud-wat-dev
      path: runs/test/Muncie/Muncie.x04
  - id: cb950b3f-e4ff-4109-936c-aaaaaaaaaaa5
    filename: Muncie.b04
    resource_info:
      store: s3
      root: cloud-wat-dev
      path: runs/test/Muncie/Muncie.c04
outputs:
  - filename: Muncie.p04.hdf
    resource_info:
      store: s3
      root: cloud-wat-dev
      path: runs/test/results/Muncie/Muncie.p04.hdf
  - filename: Muncie.b04
    resource_info:
      store: s3
      root: cloud-wat-dev
      path: runs/test/results/Muncie/Muncie.p04.log
```
