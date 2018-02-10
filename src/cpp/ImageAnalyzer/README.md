Obtain geometry file:
cmsRun
/cvmfs/cms.cern.ch/slc6_amd64_gcc530/cms/cmssw/CMSSW_8_0_19/src/Fireworks/Geometry/python/dumpRecoGeometry_cfg.py
tag=2015 out=geom2015.root

### build pngwriter
We use [PNGwriter](http://pngwriter.sourceforge.net/) package to create PNG
images of CMS events. To build this package please download it from
[here](https://github.com/pngwriter/pngwriter) and build as following:
```
# cd into your CMSSW area and invoke cmsenv which will setup proper environment
cmsenv

# cd into pngwriter area to build the package
cd pngwriter

mkdir -p pngwriter-build
cd pngwriter-build

# install package into your INTSALL_PATH and build shared library
cmake -DCMAKE_INSTALL_PREFIX=$INSTALL_PATH -DBUILD_SHARED_LIBS=ON ../

# run make to compile the code
make -j 8

# install the code
make install
```

Now we need to write complementary CMSSW xml file to include this package into
CMSSW build. Create new pngwriter.xml file with the folliwung content (please change
paths to your own INSTALL path):
```
<tool name="pngwriter" version="1.0">
  <client>
      <environment name="PNGWRITER_BASE" default="/afs/cern.ch/user/v/valya/workspace/soft/usr"/>
      <environment name="LIBDIR" default="/afs/cern.ch/user/v/valya/workspace/soft/usr/lib"/>
      <environment name="INCLUDE" default="/afs/cern.ch/user/v/valya/workspace/soft/usr/include"/>
  </client>
  <lib name="PNGwriter"/>
  <flags CXXFLAGS="-DNO_FREETYPE"/>
</tool>
```
Once thie file is created run the following command to instruct scram to use
it:
```
scram setup pngwriter.xml
```

Finally, you'll need to modify your plugins/BuildFile.xml to add these lines:
```
<use name="pngwriter"/>
<flags EDM_PLUGIN="1"/>
<flags CppDefines="NO_FREETYPE"/>
```

Here is a full example of BuildFile.xml to use with pngwriter
```
<use name="FWCore/Framework"/>
<use name="Fireworks/Geometry"/>
<use name="Fireworks/Tracks"/>
<use name="FWCore/PluginManager"/>
<use name="FWCore/ParameterSet"/>
<use name="DataFormats/TrackReco"/>
<use name="DataFormats/TrackerRecHit2D"/>
<use name="CommonTools/UtilAlgos"/>
<use name="pngwriter"/>
<flags EDM_PLUGIN="1"/>
<flags CppDefines="NO_FREETYPE"/>
```
