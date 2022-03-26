FROM registry.access.redhat.com/ubi8/ubi:8.5 as build
RUN yum -y update && \ 
    yum -y install wget && \
    yum -y install unzip

RUN wget https://www.hec.usace.army.mil/software/hec-ras/downloads/HEC-RAS_610_Linux.zip

RUN unzip HEC-RAS_610_Linux.zip
RUN unzip HEC-RAS_610_Linux/RAS_Linux_test_setup.zip 

FROM registry.access.redhat.com/ubi8/ubi:8.5 as prod
COPY --from=build /RAS_Linux_test_setup/libs /ras/libs
COPY --from=build /RAS_Linux_test_setup/Ras_v61/Release /ras/v61

RUN chmod +x ras/v61/*

RUN mkdir /sim 
WORKDIR /sim
COPY run-model.sh .
RUN chmod +x run-model.sh