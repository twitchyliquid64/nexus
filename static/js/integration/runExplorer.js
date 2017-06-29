app.controller('IntegrationRunExplorer', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.runnable = null;
  $scope.runs = [];
  $scope.error = null;
  $scope.limit = 200;

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
       console.log($scope.runs);
       $scope.updateEntries();
     }, function errorCallback(response) {
       $scope.loading = false;
       $scope.error = response;
     });
  }

  $scope.updateEntries = function(){
    $scope.loading = true;
    $scope.error = null;
    var d = {
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
      out += "<div class='log-class'>" + lines[i].Kind + "</div>";
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
  });

  $rootScope.$on('page-change', function(event, args) {
    if (args.page == 'integration-run-explorer'){
      $scope.update();
    }
  });

}]);
