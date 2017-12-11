// -*- C++ -*-
//
// Package:    Demo/TFModelAnalyzer
// Class:      TFModelAnalyzer
// 
/**\class TFModelAnalyzer TFModelAnalyzer.cc Demo/TFModelAnalyzer/plugins/TFModelAnalyzer.cc

 Description: [one line class summary]

 Implementation:
     [Notes on implementation]
*/
//
// Original Author:  Valentin Y Kuznetsov
//         Created:  Wed, 03 Aug 2016 19:31:32 GMT
//
//
// Code examples used to write this code
// https://curl.haxx.se/libcurl/c/simple.html


// system include files
#include <memory>

// curl
#include <curl/curl.h>

// user include files
#include "FWCore/Framework/interface/Frameworkfwd.h"
#include "FWCore/Framework/interface/one/EDAnalyzer.h"

#include "FWCore/Framework/interface/Event.h"
#include "FWCore/Framework/interface/MakerMacros.h"

#include "FWCore/ParameterSet/interface/ParameterSet.h"

// track includes
#include "DataFormats/TrackReco/interface/Track.h"
#include "DataFormats/TrackReco/interface/TrackFwd.h"
#include "FWCore/MessageLogger/interface/MessageLogger.h"

// c2numpy convertion include
#include "Demo/TFModelAnalyzer/interface/c2numpy.h"

// tfaas protobuf
#include "Demo/TFModelAnalyzer/interface/tfaas.pb.h"

// fireworks and geometry includes
#include "TEveGeoShape.h"
#include "TEvePointSet.h"

#include "Fireworks/Core/interface/FWSimpleProxyBuilderTemplate.h"
#include "Fireworks/Core/interface/FWGeometry.h"
#include "Fireworks/Core/interface/FWEventItem.h"
#include "Fireworks/Tracks/interface/TrackUtils.h"
#include "Fireworks/Core/interface/fwLog.h"

#include "DataFormats/TrackerRecHit2D/interface/SiPixelRecHit.h"
#include "DataFormats/TrackerRecHit2D/interface/SiStripRecHit2D.h"
#include "DataFormats/TrackerRecHit2D/interface/SiStripRecHit1D.h"

#include <iostream>
using namespace std;

// example from libcurl

struct MemoryStruct {
  char *memory;
  size_t size;
};
 
static size_t
WriteMemoryCallback(void *contents, size_t size, size_t nmemb, void *userp)
{
  size_t realsize = size * nmemb;
  struct MemoryStruct *mem = (struct MemoryStruct *)userp;
 
  mem->memory = (char*)realloc(mem->memory, mem->size + realsize + 1);
  if(mem->memory == NULL) {
    /* out of memory! */ 
    std::cout << "not enough memory (realloc returned NULL)" << std::endl;
    return 0;
  }
 
  memcpy(&(mem->memory[mem->size]), contents, realsize);
  mem->size += realsize;
  mem->memory[mem->size] = 0;
 
  return realsize;
}
// helper function
void ReadData(std::string url);
void ReadData(std::string url) {
    // read some URL
    CURL *curl = NULL;
    CURLcode res;
    struct MemoryStruct chunk;
    chunk.memory = (char*)malloc(1);  /* will be grown as needed by the realloc above */ 
    chunk.size = 0;    /* no data at this point */ 

    curl_global_init(CURL_GLOBAL_ALL);
    curl = curl_easy_init();
    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);

    // send all data to this function
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteMemoryCallback);
     
    // we pass our 'chunk' struct to the callback function
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, (void *)&chunk);
     
    // some servers don't like requests that are made without user-agent field, so we provide one
    curl_easy_setopt(curl, CURLOPT_USERAGENT, "TFModelAnalyzer-libcurl-agent");

    res = curl_easy_perform(curl);
    if(res != CURLE_OK) {
        std::cout << "Failed to get: " << url << ", error " << curl_easy_strerror(res) << std::endl;
    } else {
        std::cout << chunk.size << " bytes retrieved" << endl;
    }
    // clean-up after we done with curl calls
    curl_easy_cleanup(curl);
    free(chunk.memory);
    curl_global_cleanup();
}


// local function to get Pixel Detector Hits, see
// http://cmslxr.fnal.gov/source/Fireworks/Tracks/src/TrackUtils.cc#0611
void
pixelHits( std::vector<TVector3> &pixelPoints, FWGeometry *geom, const reco::Track &t );
void
pixelHits( std::vector<TVector3> &pixelPoints, FWGeometry *geom, const reco::Track &t )
{       
   for( trackingRecHit_iterator it = t.recHitsBegin(), itEnd = t.recHitsEnd(); it != itEnd; ++it )
   {
      const TrackingRecHit* rh = &(**it);           
      // -- get position of center of wafer, assuming (0,0,0) is the center
      DetId id = (*it)->geographicalId();
      if( ! geom->contains( id ))
      {
         fwLog( fwlog::kError )
            << "failed to get geometry of Tracker Det with raw id: " 
            << id.rawId() << std::endl;

        continue;
      }

      // -- in which detector are we?           
      unsigned int subdet = (unsigned int)id.subdetId();
            
      if(( subdet == PixelSubdetector::PixelBarrel ) || ( subdet == PixelSubdetector::PixelEndcap ))
      {
         if( const SiPixelRecHit* pixel = dynamic_cast<const SiPixelRecHit*>( rh ))
         {
            const SiPixelCluster& c = *( pixel->cluster());
            fireworks::pushPixelCluster( pixelPoints, *geom, id, c, geom->getParameters( id ));
         } 
      }
   }
}

const SiStripCluster* extractClusterFromTrackingRecHit( const TrackingRecHit* rechit );
const SiStripCluster* extractClusterFromTrackingRecHit( const TrackingRecHit* rechit )
{
   const SiStripCluster* cluster = 0;

   if( const SiStripRecHit2D* hit2D = dynamic_cast<const SiStripRecHit2D*>( rechit ))
   {     
	 cluster = hit2D->cluster().get();
   }
   if( cluster == 0 )
   {
     if( const SiStripRecHit1D* hit1D = dynamic_cast<const SiStripRecHit1D*>( rechit ))
     {
	   cluster = hit1D->cluster().get();
     }
   }
   return cluster;
}
void
SiStripClusters( std::vector<TVector3> &points, FWGeometry *geom, const reco::Track &t );
void
SiStripClusters( std::vector<TVector3> &points, FWGeometry *geom, const reco::Track &t )
{       
   bool addNearbyClusters = true;
   const edmNew::DetSetVector<SiStripCluster> * allClusters = 0;
   if( addNearbyClusters )
   {
      for( trackingRecHit_iterator it = t.recHitsBegin(), itEnd = t.recHitsEnd(); it != itEnd; ++it )
      {
         if( typeid( **it ) == typeid( SiStripRecHit2D ))
         {
            const SiStripRecHit2D &hit = static_cast<const SiStripRecHit2D &>( **it );
            if( hit.cluster().isNonnull() && hit.cluster().isAvailable()) {
               edm::Handle<edmNew::DetSetVector<SiStripCluster> > allClustersHandle;
               allClusters = allClustersHandle.product();
               break;
            }
         }
         else if( typeid( **it ) == typeid( SiStripRecHit1D ))
         {
            const SiStripRecHit1D &hit = static_cast<const SiStripRecHit1D &>( **it );
            if( hit.cluster().isNonnull() && hit.cluster().isAvailable())
            {
               edm::Handle<edmNew::DetSetVector<SiStripCluster> > allClustersHandle;
               allClusters = allClustersHandle.product();
               break;
            }
         }
      }
   }

   for( trackingRecHit_iterator it = t.recHitsBegin(), itEnd = t.recHitsEnd(); it != itEnd; ++it )
   {
      unsigned int rawid = (*it)->geographicalId();
      if( ! geom->contains( rawid ))
      {
         fwLog( fwlog::kError )
           << "failed to get geometry of SiStripCluster with detid: " 
           << rawid << std::endl;
	 
         continue;
      }
	
      const float* pars = geom->getParameters( rawid );
      
      // -- get phi from SiStripHit
      auto rechitRef = *it;
      const TrackingRecHit *rechit = &( *rechitRef );
      const SiStripCluster *cluster = extractClusterFromTrackingRecHit( rechit );

      if( cluster )
      {
         if( allClusters != 0 )
         {
            const edmNew::DetSet<SiStripCluster> & clustersOnThisDet = (*allClusters)[rechit->geographicalId().rawId()];

            for( edmNew::DetSet<SiStripCluster>::const_iterator itc = clustersOnThisDet.begin(), edc = clustersOnThisDet.end(); itc != edc; ++itc )
            {

               short firststrip = itc->firstStrip();

               float localTop[3] = { 0.0, 0.0, 0.0 };
               float localBottom[3] = { 0.0, 0.0, 0.0 };

               fireworks::localSiStrip( firststrip, localTop, localBottom, pars, rawid );

               float globalTop[3];
               float globalBottom[3];
               geom->localToGlobal( rawid, localTop, globalTop, localBottom, globalBottom );

               TVector3 pt( globalTop[0], globalTop[1], globalTop[2] );
               points.push_back( pt );
               TVector3 pb( globalBottom[0], globalBottom[1], globalBottom[2] );
               points.push_back( pb );
      
            }
         }
         else
         {
            short firststrip = cluster->firstStrip();
            
            float localTop[3] = { 0.0, 0.0, 0.0 };
            float localBottom[3] = { 0.0, 0.0, 0.0 };

            fireworks::localSiStrip( firststrip, localTop, localBottom, pars, rawid );

            float globalTop[3];
            float globalBottom[3];
            geom->localToGlobal( rawid, localTop, globalTop, localBottom, globalBottom );

               TVector3 pt( globalTop[0], globalTop[1], globalTop[2] );
               points.push_back( pt );
               TVector3 pb( globalBottom[0], globalBottom[1], globalBottom[2] );
               points.push_back( pb );
      
      
         }		
      }
      else if( !rechit->isValid() && ( rawid != 0 )) // lost hit
      {
         if( allClusters != 0 )
         {
            edmNew::DetSetVector<SiStripCluster>::const_iterator itds = allClusters->find( rawid );
            if( itds != allClusters->end())
            {
               const edmNew::DetSet<SiStripCluster> & clustersOnThisDet = *itds;
               for( edmNew::DetSet<SiStripCluster>::const_iterator itc = clustersOnThisDet.begin(), edc = clustersOnThisDet.end(); itc != edc; ++itc )
               {
                  short firststrip = itc->firstStrip();

                  float localTop[3] = { 0.0, 0.0, 0.0 };
                  float localBottom[3] = { 0.0, 0.0, 0.0 };

                  fireworks::localSiStrip( firststrip, localTop, localBottom, pars, rawid );

                  float globalTop[3];
                  float globalBottom[3];
                  geom->localToGlobal( rawid, localTop, globalTop, localBottom, globalBottom );

                   TVector3 pt( globalTop[0], globalTop[1], globalTop[2] );
                   points.push_back( pt );
                   TVector3 pb( globalBottom[0], globalBottom[1], globalBottom[2] );
                   points.push_back( pb );
          
               }
            }
         }
      }
      else
      {
         fwLog( fwlog::kDebug )
            << "*ANOTHER* option possible: valid=" << rechit->isValid()
            << ", rawid=" << rawid << std::endl;
      }
   }
}

/*
// local function to get SiStripClusters, see
// http://cmslxr.fnal.gov/source/Fireworks/Tracks/src/TrackUtils.cc#0422
void
SiStripClusters( std::vector<TVector3> &points, FWGeometry *geom, const reco::Track &t );
void
SiStripClusters( std::vector<TVector3> &points, FWGeometry *geom, const reco::Track &t )
{       
//   const edmNew::DetSetVector<SiStripCluster> * allClusters = 0;
   for( trackingRecHit_iterator it = t.recHitsBegin(), itEnd = t.recHitsEnd(); it != itEnd; ++it )
   {
       unsigned int rawid = (*it)->geographicalId();
       if( ! geom->contains( rawid ))
       {
          fwLog( fwlog::kError )
            << "failed to get geometry of SiStripCluster with detid: " 
            << rawid << std::endl;
          
          continue;
       }
     
       const float* pars = geom->getParameters( rawid );
       
       // -- get phi from SiStripHit
       auto rechitRef = *it;
       const TrackingRecHit *rechit = &( *rechitRef );

       const SiStripCluster *cluster = fireworks::extractClusterFromTrackingRecHit( rechit );
       if (cluster)
       {
           short firststrip = cluster->firstStrip();
           float localTop[3] = { 0.0, 0.0, 0.0 };
           float localBottom[3] = { 0.0, 0.0, 0.0 };
 
           fireworks::localSiStrip( firststrip, localTop, localBottom, pars, rawid );
 
           float globalTop[3];
           float globalBottom[3];
           geom->localToGlobal( rawid, localTop, globalTop, localBottom, globalBottom );
           TVector3 pt( globalTop[0], globalTop[1], globalTop[2] );
           points.push_back( pt );
           TVector3 pb( globalBottom[0], globalBottom[1], globalBottom[2] );
           points.push_back( pb );
       }
   }
}
*/

//
// class declaration
//

// If the analyzer does not use TFileService, please remove
// the template argument to the base class so the class inherits
// from  edm::one::EDAnalyzer<> and also remove the line from
// constructor "usesResource("TFileService");"
// This will improve performance in multithreaded jobs.

class TFModelAnalyzer : public edm::one::EDAnalyzer<edm::one::SharedResources>  {
   public:
      explicit TFModelAnalyzer(const edm::ParameterSet&);
      ~TFModelAnalyzer();

      static void fillDescriptions(edm::ConfigurationDescriptions& descriptions);


   private:
      virtual void beginJob() override;
      virtual void analyze(const edm::Event&, const edm::EventSetup&) override;
      virtual void endJob() override;

      // ----------member data ---------------------------
      // c2numpy
      c2numpy_writer writer;

      // FWGeometry
      FWGeometry *geom;

      // hits constrains
      int max_pxhits = 5;
      int max_sihits = 50;
};

//
// constants, enums and typedefs
//

//
// static data member definitions
//

//
// constructors and destructor
//
TFModelAnalyzer::TFModelAnalyzer(const edm::ParameterSet& iConfig)
{

   auto url = std::string("www.google.com");
   ReadData(url);

   //now do what ever initialization is needed
   // usesResource("TFileService");
   //
   // load geometry
   geom = new FWGeometry();
   const char* geomFile="/afs/cern.ch/user/v/valya/workspace/CMSSW/CMSSW_8_0_21/src/geom.root";
   cout << "Read geometry from " << geomFile << endl;
   geom->loadMap(geomFile);

   // c2numpy
   c2numpy_init(&writer, "output/trackparams", 1000);

   c2numpy_addcolumn(&writer, "run", C2NUMPY_INTC);
   c2numpy_addcolumn(&writer, "evt", C2NUMPY_INTC);
   c2numpy_addcolumn(&writer, "lumi", C2NUMPY_INTC);
   c2numpy_addcolumn(&writer, "TrackId", C2NUMPY_INTC);
   c2numpy_addcolumn(&writer, "charge", C2NUMPY_INTC);

   c2numpy_addcolumn(&writer, "chi2", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "ndof", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "normalizedChi2", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "qoverp", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "theta", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "lambda", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "dxy", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "d0", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "dsz", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "dz", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "p", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "pt", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "px", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "py", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "pz", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "eta", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "phi", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "vx", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "vy", C2NUMPY_FLOAT64);
   c2numpy_addcolumn(&writer, "vz", C2NUMPY_FLOAT64);

   for (auto i = 0;  i < max_pxhits;  ++i) { // number of pixel hits
       std::ostringstream name;
       name << "pix_" << i;
       for (auto j = 0;  j < 3;  ++j) { // 3 coordinates
           std::ostringstream cname;
           if(j==0) cname << name.str() << "_x";
           if(j==1) cname << name.str() << "_y";
           if(j==2) cname << name.str() << "_z";
           c2numpy_addcolumn(&writer, cname.str().c_str(), C2NUMPY_FLOAT64);
       }
   }

   for (auto i = 0;  i < max_sihits;  ++i) { // number of sistrip clusters
       std::ostringstream name;
       name << "sis_" << i;
       for (auto j = 0;  j < 3;  ++j) { // 3 coordinates
           std::ostringstream cname;
           if(j==0) cname << name.str() << "_x";
           if(j==1) cname << name.str() << "_y";
           if(j==2) cname << name.str() << "_z";
           c2numpy_addcolumn(&writer, cname.str().c_str(), C2NUMPY_FLOAT64);
       }
   }

   // a module register what data it will request from the Event, Chris' suggestion
   consumes<reco::TrackCollection>(edm::InputTag("generalTracks"));

}


TFModelAnalyzer::~TFModelAnalyzer()
{
 
   // do anything here that needs to be done at desctruction time
   // (e.g. close files, deallocate resources etc.)
   geom->clear();
   delete geom;

}


//
// member functions
//

// ------------ method called for each event  ------------
void
TFModelAnalyzer::analyze(const edm::Event& iEvent, const edm::EventSetup& iSetup)
{
   using namespace edm;


   Handle<reco::TrackCollection> tracks;
   iEvent.getByLabel("generalTracks", tracks); 
   LogInfo("Demo") << "number of tracks "<<tracks->size();

   // get event id
   auto eid = iEvent.id();

   // c2numpy
   int tidx = 0;
   for (auto track = tracks->cbegin();  track != tracks->end();  ++track, ++tidx) {
       // extract track parameters

       c2numpy_intc(&writer, eid.run());
       c2numpy_intc(&writer, eid.event());
       c2numpy_intc(&writer, eid.luminosityBlock());
       c2numpy_intc(&writer, tidx);
       c2numpy_intc(&writer, track->charge());

       c2numpy_float64(&writer, track->chi2());
       c2numpy_float64(&writer, track->ndof());
       c2numpy_float64(&writer, track->normalizedChi2());
       c2numpy_float64(&writer, track->qoverp());
       c2numpy_float64(&writer, track->theta());
       c2numpy_float64(&writer, track->lambda());
       c2numpy_float64(&writer, track->dxy());
       c2numpy_float64(&writer, track->d0());
       c2numpy_float64(&writer, track->dsz());
       c2numpy_float64(&writer, track->dz());
       c2numpy_float64(&writer, track->p());
       c2numpy_float64(&writer, track->pt());
       c2numpy_float64(&writer, track->px());
       c2numpy_float64(&writer, track->py());
       c2numpy_float64(&writer, track->pz());
       c2numpy_float64(&writer, track->eta());
       c2numpy_float64(&writer, track->phi());
       c2numpy_float64(&writer, track->vx());
       c2numpy_float64(&writer, track->vy());
       c2numpy_float64(&writer, track->vz());

       // extract Pixel hits
       int npxhits = 0;
       std::vector<TVector3> pxpoints;
       pixelHits( pxpoints, geom, *track );
       cout << "pixel hits" << endl;
       for( auto it = pxpoints.begin(), itEnd = pxpoints.end(); it != itEnd; ++it, ++npxhits) {
           //cout << " x=" << it->x() << " y=" << it->y() << " z=" << it->z() << endl;
           cout << it->x() << "," << it->y() << "," << it->z() << endl;
           if (npxhits<max_pxhits) {
               c2numpy_float64(&writer, it->x());
               c2numpy_float64(&writer, it->y());
               c2numpy_float64(&writer, it->z());
           }
       }
       // fill the rest
       for(auto i=npxhits; i < max_pxhits; ++i){
//           cout << "x=0 y=0 z=0" << endl;
           c2numpy_float64(&writer, 0.); // init x
           c2numpy_float64(&writer, 0.); // init y
           c2numpy_float64(&writer, 0.); // init z
       }

       // extract SiStripClusters
       int nsihits = 0;
       std::vector<TVector3> sipoints;
       SiStripClusters(sipoints, geom, *track);
       cout << "SiStrip clusters" << endl;
       for( auto it = sipoints.begin(), itEnd = sipoints.end(); it != itEnd; ++it, ++nsihits) {
           //cout << " x=" << it->x() << " y=" << it->y() << " z=" << it->z() << endl;
           cout << it->x() << "," << it->y() << "," << it->z() << endl;
           if (nsihits<max_sihits) {
               c2numpy_float64(&writer, it->x());
               c2numpy_float64(&writer, it->y());
               c2numpy_float64(&writer, it->z());
           }
       }
       // fill the rest
       for(auto i=nsihits; i < max_sihits; ++i){
           c2numpy_float64(&writer, 0.); // init x
           c2numpy_float64(&writer, 0.); // init y
           c2numpy_float64(&writer, 0.); // init z
       }
       LogInfo("Track") << tidx << "charge " << track->charge() << "pt " << track->pt() << "# pixel hits " << npxhits << "# sihits " << nsihits;

   }

}


// ------------ method called once each job just before starting event loop  ------------
void 
TFModelAnalyzer::beginJob()
{
}

// ------------ method called once each job just after ending the event loop  ------------
void 
TFModelAnalyzer::endJob() 
{
  // c2numpy
  c2numpy_close(&writer);
}

// ------------ method fills 'descriptions' with the allowed parameters for the module  ------------
void
TFModelAnalyzer::fillDescriptions(edm::ConfigurationDescriptions& descriptions) {
  //The following says we do not know what parameters are allowed so do no validation
  // Please change this to state exactly what you do use, even if it is no parameters
  edm::ParameterSetDescription desc;
  desc.setUnknown();
  descriptions.addDefault(desc);
}

//define this as a plug-in
DEFINE_FWK_MODULE(TFModelAnalyzer);
