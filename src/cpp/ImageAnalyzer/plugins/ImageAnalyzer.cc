// -*- C++ -*-
//
// Package:    Image/ImageAnalyzer
// Class:      ImageAnalyzer
// 
/**\class ImageAnalyzer ImageAnalyzer.cc Image/ImageAnalyzer/plugins/ImageAnalyzer.cc

 Description: [one line class summary]

 Implementation:
     [Notes on implementation]
*/
//
// Original Author:  Valentin Y Kuznetsov
//         Created:  Wed, 03 Aug 2016 19:31:32 GMT
//

// system include files
#include <memory>
#include <stdlib.h>

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

// local includes
#include "Image/ImageAnalyzer/interface/pngwriter.h"


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

// Example how to make a PNG file
void
makePNG( pngwriter &png, int width, int height, const std::string &fname, std::vector<TVector3> &points, double red, double green, double blue);
void
makePNG( pngwriter &png, int width, int height, const std::string &fname, std::vector<TVector3> &points, double red, double green, double blue)
{
    int hdim = height/3;
    // detector boundaries
    double bmaxx = 120;
    double bminx =-120;
    double bmaxy = 120;
    double bminy =-120;
    double bmaxz = 300;
    double bminz =-300;
    // picture boundaries
    double maxx = hdim; // e.g. 100
    double maxy = hdim; // e.g. 100
    double maxz = width; // e.g. 300
    for( auto it = points.begin(), itEnd = points.end(); it != itEnd; ++it) {
        // normalize to detector boundaries: (v-min_v)/(max_v-min_v)
        auto z = (it->z()-bminz)/(bmaxz-bminz); // max_v=maxz, min_v=-maxz
        auto x = (it->x()-bminx)/(bmaxx-bminx);
        auto y = (it->y()-bminy)/(bmaxy-bminy);
        // normalize to picture boundaries
        z = z*maxz;
        x = x*maxx;
        y = y*maxy;
        // xz projection (z along width)
        if(z <= width && z >= 1 && x <= hdim && x >= 1) {
            png.plot((int)z, (int)x, red, green, blue);
        }
        // xy projection (x along width)
        if(x <= width && x >= 1 && y <= hdim && y >= 1) {
            png.plot((int)x, (int)(hdim+y), red, green, blue);
        }
        // yz projection (z along width)
        if(z <= width && z >= 1 && y <= hdim && y >= 1) {
            png.plot((int)z, (int)(2*hdim+y), red, green, blue);
        }
    }
}
/*
void
makePNG( int width, int height, const std::string &fname, std::vector<TVector3> &points);
void
makePNG( int width, int height, const std::string &fname, std::vector<TVector3> &points)
{
   pngwriter png( width, height*3, 0, fname.c_str());
   int red = 1;
   int green = 0;
   int blue = 0;
   for( auto it = points.begin(), itEnd = points.end(); it != itEnd; ++it) {
       // xz projection
       if(abs(it->x()) < width && abs(it->z()) < height/3) {
           png.plot(it->x(), it->z(), red, green, blue);
       }
       // xy projection
       if(abs(it->x()) < width && abs(it->y()) < height/3) {
           png.plot(it->x(), height+it->y(), red, green, blue);
       }
       // yz projection
       if(abs(it->y()) < width && abs(it->z()) < height/3) {
           png.plot(it->y(), 2*height+it->z(), red, green, blue);
       }
   }
   png.close();
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

class ImageAnalyzer : public edm::one::EDAnalyzer<edm::one::SharedResources>  {
   public:
      explicit ImageAnalyzer(const edm::ParameterSet&);
      ~ImageAnalyzer();

      static void fillDescriptions(edm::ConfigurationDescriptions& descriptions);


   private:
      virtual void beginJob() override;
      virtual void analyze(const edm::Event&, const edm::EventSetup&) override;
      virtual void endJob() override;

      // ----------member data ---------------------------
      // FWGeometry
      FWGeometry *geom;

      // output directory where we store results (png's and numpy files)
      std::string outputDir;

      // png dimensions
      int png_width;
      int png_height;
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
ImageAnalyzer::ImageAnalyzer(const edm::ParameterSet& iConfig)

{
   // parse configuration parameter set
   outputDir = iConfig.retrieveUntracked("outputDir")->getString();
   png_width = iConfig.retrieveUntracked("pngWidth")->getInt32();
   png_height = iConfig.retrieveUntracked("pngHeight")->getInt32();
   // load geometry
   geom = new FWGeometry();
   auto geomFile = iConfig.retrieveUntracked("geomFile")->getString();
   cout << "Read geometry from " << geomFile << endl;
   geom->loadMap(geomFile.c_str());

   // a module register what data it will request from the Event, Chris' suggestion
   consumes<reco::TrackCollection>(edm::InputTag("generalTracks"));

}


ImageAnalyzer::~ImageAnalyzer()
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
ImageAnalyzer::analyze(const edm::Event& iEvent, const edm::EventSetup& iSetup)
{
   using namespace edm;


   Handle<reco::TrackCollection> tracks;
   iEvent.getByLabel("generalTracks", tracks); 
   LogInfo("Image") << "number of tracks "<<tracks->size();

   // get event id
   auto eid = iEvent.id();

   // create PNG image of the event, we'll use in a single plot
   // 3 projections (xz, xy, yz) therefore we stack them as height*3
   std::string fname = outputDir+"/run"+std::to_string(eid.run())+"_evt"+std::to_string(eid.event())+"_lumi"+std::to_string(eid.luminosityBlock())+".png"; 
   auto height = png_height*3;
   auto width = png_width;
   pngwriter png( width, height, 0, fname.c_str());
   double red, green, blue;

   // loop over tracks to extract hits
   int tidx = 0;
   std::vector<TVector3> hits;
   for (auto track = tracks->cbegin();  track != tracks->end();  ++track, ++tidx) {
       // extract Pixel hits
       int npxhits = 0;
       std::vector<TVector3> pxpoints;
       pixelHits( pxpoints, geom, *track );
       cout << "pixel hits" << endl;
       for( auto it = pxpoints.begin(), itEnd = pxpoints.end(); it != itEnd; ++it, ++npxhits) {
           cout << it->x() << "," << it->y() << "," << it->z() << endl;
       }
       red = 1.0;
       green = 0.0;
       blue = 0.0;
       makePNG(png, width, height, fname, pxpoints, red, green, blue);

       // extract SiStripClusters
       int nsihits = 0;
       std::vector<TVector3> sipoints;
       SiStripClusters(sipoints, geom, *track);
       cout << "SiStrip clusters" << endl;
       for( auto it = sipoints.begin(), itEnd = sipoints.end(); it != itEnd; ++it, ++nsihits) {
           cout << it->x() << "," << it->y() << "," << it->z() << endl;
       }
       red = 0.0;
       green = 1.0;
       blue = 0.0;
       makePNG(png, width, height, fname, sipoints, red, green, blue);
   }
   // close PNG image
   png.close();

}


// ------------ method called once each job just before starting event loop  ------------
void 
ImageAnalyzer::beginJob()
{
}

// ------------ method called once each job just after ending the event loop  ------------
void 
ImageAnalyzer::endJob() 
{
}

// ------------ method fills 'descriptions' with the allowed parameters for the module  ------------
void
ImageAnalyzer::fillDescriptions(edm::ConfigurationDescriptions& descriptions) {
  //The following says we do not know what parameters are allowed so do no validation
  // Please change this to state exactly what you do use, even if it is no parameters
  edm::ParameterSetDescription desc;
  desc.setUnknown();
  descriptions.addDefault(desc);
}

//define this as a plug-in
DEFINE_FWK_MODULE(ImageAnalyzer);
