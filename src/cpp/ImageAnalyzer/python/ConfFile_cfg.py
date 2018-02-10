import FWCore.ParameterSet.Config as cms

process = cms.Process("Image")

# initialize MessageLogger and output report
process.load("FWCore.MessageLogger.MessageLogger_cfi")
process.MessageLogger.cerr.threshold = 'INFO'
process.MessageLogger.categories.append('Image')
process.MessageLogger.cerr.INFO = cms.untracked.PSet(
    limit = cms.untracked.int32(-1)
)
#process.options   = cms.untracked.PSet( wantSummary = cms.untracked.bool(True) )

# process all events
#process.maxEvents = cms.untracked.PSet( input = cms.untracked.int32(-1) )
# process 10 events
process.maxEvents = cms.untracked.PSet( input = cms.untracked.int32(10) )

#process.add_(cms.ESProducer("FWRecoGeometryESProducer"))

process.source = cms.Source("PoolSource",
    # replace 'myfile.root' with the source file you want to use
    fileNames = cms.untracked.vstring(
#        'file:/afs/cern.ch/user/v/valya/workspace/CMSSW/data/RelValJpsiMuMu_Pt-8_GEN-SIM-RECO.root'
        'file:/afs/cern.ch/user/v/valya/workspace/CMSSW/data/RelValHiggs200ChartedTaus13_GEN-SIM-RECO.root'
    )
)

process.dump=cms.EDAnalyzer('EventContentAnalyzer')
process.demo = cms.EDAnalyzer('ImageAnalyzer',
    pngWidth = cms.untracked.int32(300),
    pngHeight = cms.untracked.int32(100),
    outputDir = cms.untracked.string('/afs/cern.ch/user/v/valya/workspace/CMSSW/output'),
    geomFile = cms.untracked.string('/afs/cern.ch/user/v/valya/workspace/CMSSW/data/geom.root'))

#process.p = cms.Path(process.demo)
process.p = cms.Path(process.demo*process.dump)
