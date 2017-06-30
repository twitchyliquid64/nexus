app.controller('IntegrationRunExplorer', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.runnable = null;
  $scope.runs = [];
  $scope.run = '!!';
  $scope.error = null;
  $scope.limit = 200;
  $scope.filters = {};

  $scope.update = function(){
    $scope.loading = true;
    $scope.error = null;
     $http({
       method: 'POST',
       url: '/web/v1/integrations/log/runs',
       data: [$scope.runnable.UID],
     }).then(function successCallback(response) {
       $scope.loading = false;
       $scope.runs = response.data;
       $scope.updateEntries();
     }, function errorCallback(response) {
       $scope.loading = false;
       $scope.error = response;
     });
  }

  $scope.updateEntries = function(){
    if ($scope.startRun){
      $scope.run = $scope.startRun;
      $scope.startRun = undefined;
      $scope.$broadcast('run-filter-update-run', {run: $scope.run});
    }

    $scope.loading = true;
    $scope.error = null;
    var d = {
      Sys: $scope.filters.sys,
      Problem: $scope.filters.prob,
      Info: $scope.filters.info,
      RunnableUID: $scope.runnable.UID,
      Offset: $scope.offset || 0,
      Limit: $scope.limit,
    }
    if ($scope.run && $scope.run != '!!') {
      d.RunID = $scope.run;
    }
     $http({
       method: 'POST',
       url: '/web/v1/integrations/log/entries',
       data: d,
     }).then(function successCallback(response) {
       $scope.loading = false;
       console.log(response.data);
       document.getElementById('integrationLogOutput').innerHTML = logLinesToHtml(response.data || []);
     }, function errorCallback(response) {
       $scope.loading = false;
       $scope.error = response;
     });
  }

  function logLinesToHtml(lines){
    var out = "<div class='log-container'>";
    for (var i = 0; i < lines.length; i++) {
      var d = moment(lines[i].CreatedAt);

      out += "<div class='log-line'>";
      out += "<div class='log-date'>" + d.format("HH:mm:ss L") + "</div>";

      out += "<div class='log-class'>";
      if (lines[i].Level == 1){
        out += "<div class='log-cell lda'>W</div>";
      } else if (lines[i].Level == 2){
        out += "<div class='log-cell ldr'>E</div>";
      } else {
        out += "<div class='log-cell'>I</div>";
      }

      if (lines[i].Kind == 'control'){
        out += "<div class='log-cell ldg'>C</div>";
      } else if (lines[i].Kind == 'data'){
        out += "<div class='log-cell ldl'>D</div>";
      } else {
        out += "<div class='log-cell'></div>";
      }

      if (lines[i].Datatype == 1){
        out += "<div class='log-cell ldg'>S</div>";
      } else if (lines[i].Datatype == 2){
        out += "<div class='log-cell'>I</div>";
      } else if (lines[i].Datatype == 3){
        out += "<div class='log-cell ldi'>S</div>";
      } else if (lines[i].Datatype == 4){
        out += "<div class='log-cell ldi'>E</div>";
      } else {
        out += "<div class='log-cell'></div>";
      }
      out += "</div>";

      out += "<div class='log-content'>" + lines[i].Value + "</div>";
      out += "</div>";
    }
    out += "</div>";
    return out;
  }

  $scope.filtersChanged = function(run, filters, limit, offset){
    $scope.filters = filters;
    $scope.limit = parseInt(limit);
    $scope.offset = parseInt(offset);
    $scope.run = run;
    console.log("New constraints: ", run, filters, limit, offset);
    $scope.updateEntries();
  }

  $rootScope.$on('integration-run-explorer', function(event, args) {
    $scope.runnable = args.runnable;
    $scope.runs = [];
    if (args.runID){
      $scope.startRun = args.runID;
    } else {
      $scope.startRun = null;
    }
  });

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'integration-run-explorer'){
      $scope.update();
    }
  });

}]);
