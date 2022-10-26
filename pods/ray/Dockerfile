FROM registry.access.redhat.com/ubi8/python-38:1-100
# the base image changes the user to a non root user
USER root 
# create directory
RUN mkdir /downloads && chmod 777 /downloads

# install stuff && downgrade the protobuf version,
#see https://github.com/ray-project/ray/issues/25282
RUN pip install --ignore-installed "ray[default]"==2.0.0 kubernetes==v24.2.0 minio==7.1.2 protobuf==3.20.0

WORKDIR /home
# Copy code
COPY src/ray-pod.py ray-pod.py
#run the ray-pod as non privileged acount
USER 65532:65532
CMD [ "python", "-u", "./ray-pod.py" ]
