#!/bin/bash
#SBATCH --job-name=bridgetest
#SBATCH --output=bridgetest.out
module load intelmpi
echo $PWD
