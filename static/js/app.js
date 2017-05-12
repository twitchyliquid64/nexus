var app = angular.module('nexus', ['ui.materialize']);

app.controller('BodyController', ["$scope", function ($scope) {
    $scope.page = "home";
    $scope.dataService = dataService;
    $scope.changePage = function(pageName){
        $scope.page = pageName;
    };
}]);


app.controller('LoginController', ["$scope", function ($scope) {
    $scope.username = '';
    $scope.password = '';
    $scope.doLogin = function(){
      console.log($scope.username, $scope.password)
    }
}]);
