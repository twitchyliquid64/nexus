(function () {
  app.directive('runFilter', function($rootScope){
    return {
      scope: {
        runs: '=',
        runnable: '=',
        onChange: '&',
      },
      //restrict E means its can only be used as an element.
      restrict: 'E',
      templateUrl: function(elem, attr){
        return '/static/views/integration/filterDirective.html?cache=8';
      },
      link: function($scope, elem, attrs) {
        // scope = either parent scope or its own child scope if scope set.
        // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
      },
      controller: function($scope) {
        $scope.showSystem = true;
        $scope.showInfo = true;
        $scope.showProblem = true;

        $scope.selectedRun = '!!';
        $scope.maxRows = 200;
        $scope.offset = 0;

        $scope.page = 0;

        $scope.next = function(){
          $scope.page = ($scope.page + 1) % 2;
          console.log($scope.runs);
        }

        $scope.scroll = function(){
          window.scrollTo(0,document.body.scrollHeight);
        }

        $scope.add = function(){
          $scope.offset += $scope.maxRows;
          $scope.fireChange();
        }
        $scope.rem = function(){
          $scope.offset -= $scope.maxRows;
          $scope.fireChange();
        }

        $scope.fireChange = function(){
          var f = {
            info: $scope.showInfo,
            prob: $scope.showProblem,
            sys: $scope.showSystem,
          };
          $scope.onChange({run: $scope.selectedRun, filters: f, limit: $scope.maxRows, offset: $scope.offset});
        }
      },
    };
  });
})();
