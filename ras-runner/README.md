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
model_configuration:
  model_name: Muncie
model_links:
  linked_inputs:
    - name: 1
      format: object
      resource_info:
        scheme: s3
        authority: cloud-wat-dev
        fragment: runs/realization_1/event_2/Muncie.p04.tmp.hdf
    - name: 2
      format: object
      resource_info:
        scheme: s3
        authority: cloud-wat-dev
        fragment: models/Muncie/Muncie.b04
    - name: 3
      format: object
      resource_info:
        scheme: s3
        authority: cloud-wat-dev
        fragment: models/Muncie/Muncie.prj
    - name: 4
      format: object
      resource_info:
        scheme: s3
        authority: cloud-wat-dev
        fragment: models/Muncie/Muncie.x04
    - name: 5
      format: object
      resource_info:
        scheme: s3
        authority: cloud-wat-dev
        fragment: models/Muncie/Muncie.c04
  required_outputs:
    - name: 1
      format: object
      resource_info:
        scheme: s3
        authority: cloud-wat-dev
        fragment: runs/realization_1/event_2/Muncie.p04.hdf
    - name: 2
      format: object
      resource_info:
        scheme: s3
        authority: cloud-wat-dev
        fragment: runs/realization_1/event_2/Muncie.dss
    - name: 3
      format: object
      resource_info:
        scheme: s3
        authority: cloud-wat-dev
        fragment: runs/realization_1/event_2/Muncie.log
```
