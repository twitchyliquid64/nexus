var app = angular.module('nexus', ['ui.materialize', 'angularMoment']);

app.controller('BodyController', ["$scope", "$rootScope", function ($scope, $rootScope) {
    $scope.page = "home";
    $scope.changePage = function(pageName){
        $scope.page = pageName;
        $rootScope.$broadcast('page-change',{page: pageName});
    };
}]);

app.directive('loader', function($rootScope){
  return {
    scope: {
      loading: '=',
      error: '=',
    },
    //restrict E means its can only be used as an element.
    restrict: 'E',
    template: '<div class="progress" ng-show="loading"><div class="indeterminate"></div></div>  <blockquote ng-show="error"><h5>Network Error</h5>' +
        '<ul class="collection">' +
        '<li class="collection-item"><b>Error code</b>: {{error.status}}</li>' +
        '<li class="collection-item"><b>Explanation</b>: {{error.statusText}}</li>' +
        '<li class="collection-item"><b>The server said</b>: {{error.data}}</li>' +
        '</ul></blockquote>',
    link: function($scope, elem, attrs) {
      // scope = either parent scope or its own child scope if scope set.
      // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
    },
    controller: function($scope) {
    },
  };
});
