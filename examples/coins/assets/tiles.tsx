<?xml version="1.0" encoding="UTF-8"?>
<tileset version="1.10" tiledversion="1.11.0" name="tiles" tilewidth="32" tileheight="32" tilecount="8" columns="4">
 <image source="tiles.png" width="128" height="64"/>
 <tile id="0" type="Floor"/>
 <tile id="1" type="Wall"/>
 <tile id="2" type="Wall">
  <animation>
   <frame tileid="2" duration="400"/>
   <frame tileid="3" duration="400"/>
  </animation>
 </tile>
 <tile id="4" type="Player">
  <animation>
   <frame tileid="4" duration="120"/>
   <frame tileid="5" duration="120"/>
  </animation>
 </tile>
 <tile id="6" type="Coin">
  <animation>
   <frame tileid="6" duration="250"/>
   <frame tileid="7" duration="250"/>
  </animation>
 </tile>
</tileset>
