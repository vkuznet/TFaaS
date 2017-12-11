import FWCore.ParameterSet.Config as cms

process = cms.Process("TFModelDemo")

# initialize MessageLogger and output report
process.load("FWCore.MessageLogger.MessageLogger_cfi")
process.MessageLogger.cerr.threshold = 'INFO'
process.MessageLogger.categories.append('TFModelDemo')
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
        #
        'file:/afs/cern.ch/user/v/valya/workspace/CMSSW/CMSSW_8_0_21/src/C2DE82B0-D298-E611-810B-0CC47A7C3428.root'
    )
)

process.dump=cms.EDAnalyzer('EventContentAnalyzer')
process.demo = cms.EDAnalyzer('TFModelAnalyzer',
    geomFile = cms.untracked.string('/afs/cern.ch/user/v/valya/workspace/CMSSW/CMSSW_8_0_21/src/geom.root'),
    tfaasUrl = cms.untracked.string('http://localhost:8083')
)

#process.p = cms.Path(process.demo)
process.p = cms.Path(process.demo*process.dump)
