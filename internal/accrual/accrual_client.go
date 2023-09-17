package accrual

import (
	"github.com/CvitoyBamp/gopher/internal/model"
	"log"
	"time"
)

func (as *AccrualService) processOrders() {

	for {
		<-time.After(5 * time.Second)

		newOrders, errNew := as.DB.ParseAccrualByStatus(model.StatusNEW)
		if errNew != nil {
			if errNew.Error() != "no rows in result set" {
				continue
			}
			log.Print(errNew)
		}

		processingOrders, errProc := as.DB.ParseAccrualByStatus(model.StatusPROCESSING)
		if errProc != nil {
			if errProc.Error() != "no rows in result set" {
				continue
			}
			log.Print(errProc)
		}

		newOrdersChan := make(chan model.Accrual, len(newOrders))
		doneNewOrders := make(chan bool, len(newOrders))

		processingOrdersChan := make(chan model.Accrual, len(processingOrders))
		doneProcessingOrders := make(chan bool, len(processingOrders))

		errToProc := as.accrualToProcessing(newOrdersChan, doneNewOrders)
		if errToProc != nil {
			log.Print(errToProc)
		}

		errFromProc := as.accrualToRandStatus(processingOrdersChan, doneProcessingOrders)
		if errFromProc != nil {
			log.Print(errFromProc)
		}

		for _, no := range newOrders {
			newOrdersChan <- no
		}

		close(newOrdersChan)

		for range newOrders {
			<-doneNewOrders
		}

		for _, po := range processingOrders {
			processingOrdersChan <- po
		}

		close(processingOrdersChan)

		for range newOrders {
			<-doneProcessingOrders
		}

	}

}

func (as *AccrualService) accrualToProcessing(accruals <-chan model.Accrual, done chan<- bool) error {

	for accrual := range accruals {
		accrual.Status = model.StatusPROCESSING
		err := as.DB.UpdateAccrual(accrual)
		if err != nil {
			return err
		}

		done <- true
	}
	return nil
}

func (as *AccrualService) accrualToRandStatus(accruals <-chan model.Accrual, done chan<- bool) error {

	for accrual := range accruals {
		accNewStatus := as.randChangeStatus(accrual)

		err := as.DB.UpdateAccrual(accNewStatus)
		if err != nil {
			return err
		}

		done <- true
	}

	return nil
}

func (as *AccrualService) randChangeStatus(accrual model.Accrual) model.Accrual {
	acc := randAccrual()
	r := randNumber(3)

	switch r {
	case 0:
		return model.Accrual{
			Orderid: accrual.Orderid,
			Status:  model.StatusPROCESSED,
			Accrual: &acc,
		}
	case 1:
		return model.Accrual{
			Orderid: accrual.Orderid,
			Status:  model.StatusINVALID,
			Accrual: nil,
		}
	case 2:
		return model.Accrual{
			Orderid: accrual.Orderid,
			Status:  model.StatusREGISTERED,
			Accrual: nil,
		}
	default:
		return accrual
	}
}
