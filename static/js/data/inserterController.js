
app.controller('DataInserterController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.datastore = {};
  $scope.error = null;

  $scope.filtersChanged = function(filters, limit, offset){
    $scope.filters = filters;
    $scope.limit = limit;
    $scope.offset = offset;
    $scope.update();
  }

  $scope.kind = function(k){
    switch (k){
      case 0:
        return "INT";
      case 2:
        return "FLOAT";
      case 3:
        return "STR";
      case 4:
        return "BLOB";
      case 5:
        return "TIME";
    }
  }

  $scope.insert = function(){
    var cols = [];
    var outStr = "";
    for (var i = 0; i < $scope.datastore.Cols.length; i++) {
        cols[cols.length] = $scope.datastore.Cols[i].UID;
        outStr += String($scope.datastore.Cols[i].tempVal) + ",";
        $scope.datastore.Cols[i].tempVal = "";
    }

    outStr = outStr.slice(0, -1);
    $scope.loading = true;

    $http({
      method: 'POST',
      url: '/web/v1/data/insert?ds=' + $scope.datastore.UID + "&cols=" + cols.join(),
      data: outStr,
    }).then(function successCallback(response) {
      $scope.loading = false;
      $scope.error = null;
    }, function errorCallback(response) {
      $scope.loading = false;
      $scope.error = response;
    });
  }

  $rootScope.$on('data-insert', function(event, args) {
    $scope.datastore = args.ds;
  });
}]);
