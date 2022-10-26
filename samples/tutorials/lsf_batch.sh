#BSUB -J test
#BSUB -o test_%J.out
#BSUB -e test_%J.err
#BSUB -q normal
#BSUB -W 0:10
#BSUB -nnodes 1
echo $PWD
